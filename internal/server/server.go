package server

import (
	"bytes"
	"context"
	"html/template"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/rob121/cannon/internal/admin"
	"github.com/rob121/cannon/internal/api"
	"github.com/rob121/cannon/internal/blocks"
	"github.com/rob121/cannon/internal/config"
	cms "github.com/rob121/cannon/internal/content"
	"github.com/rob121/cannon/internal/groups"
	_ "github.com/rob121/cannon/internal/controllers/auth"
	_ "github.com/rob121/cannon/internal/controllers/content"
	_ "github.com/rob121/cannon/internal/notifications"
	"github.com/rob121/cannon/internal/database"
	"github.com/rob121/cannon/internal/httpx"
	"github.com/rob121/cannon/internal/install"
	"github.com/rob121/cannon/internal/lang"
	"github.com/rob121/cannon/internal/middleware"
	"github.com/rob121/cannon/internal/models"
	"github.com/rob121/cannon/internal/roles"
	"github.com/rob121/cannon/internal/router"
	"github.com/rob121/cannon/internal/routepath"
	"github.com/rob121/cannon/internal/settings"
	"github.com/rob121/cannon/internal/sites"
	"github.com/rob121/cannon/internal/templateengine"
	"github.com/rob121/cannon/internal/themes"
	"github.com/rob121/cannon/internal/user"
)

// Server is the main HTTP server.
type Server struct {
	cfg   *config.App
	sites *sites.Manager
	chain *middleware.Chain
	mux   *http.ServeMux
	mu    sync.RWMutex
}

type muxDelegate struct {
	srv *Server
}

func (d *muxDelegate) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	d.srv.mu.RLock()
	mux := d.srv.mux
	d.srv.mu.RUnlock()
	if mux != nil {
		mux.ServeHTTP(w, r)
	}
}

// New creates and wires the HTTP server.
func New(cfg *config.App) (*Server, error) {
	s := &Server{cfg: cfg, mux: http.NewServeMux()}
	if err := s.activate(); err != nil {
		return nil, err
	}
	return s, nil
}

// Activate reloads routes after install completes without restarting the process.
func (s *Server) Activate(cfg *config.App) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.cfg = cfg
	s.sites = nil
	s.chain = nil
	s.mux = http.NewServeMux()
	return s.activate()
}

// Reload reads sites.json from disk and reactivates routes, extensions, and middleware.
func (s *Server) Reload() error {
	cfg, _, err := config.Reload()
	if err != nil {
		return err
	}
	return s.Activate(cfg)
}

func (s *Server) activate() error {
	s.mux.HandleFunc("/admin/assets/", s.serveAdminThemeAsset)
	s.mux.HandleFunc("/theme/", s.serveThemeAsset)

	if config.NeedsInstall(s.cfg) {
		ih, err := install.NewHandler(s.cfg, s.Activate)
		if err != nil {
			return err
		}
		s.mux.Handle("/install", ih)
		s.mux.Handle("/install/", ih)
		s.mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			// Never send /admin traffic back to /install — that causes redirect loops
			// when install is complete but routes have not yet switched to app mode.
			if strings.HasPrefix(r.URL.Path, "/admin") {
				httpx.Redirect(w, r, "/admin/login")
				return
			}
			httpx.Redirect(w, r, "/install")
		})
		return nil
	}

	mgr, err := sites.NewManager(s.cfg)
	if err != nil {
		return err
	}
	s.sites = mgr
	s.chain = middleware.NewChain(mgr)

	for i := range s.cfg.Sites {
		site := &s.cfg.Sites[i]
		if db, err := database.Get(site.ID); err == nil {
			if err := roles.EnsureDefaults(db); err != nil {
				log.Printf("seed defaults for site %s: %v", site.ID, err)
			}
			if err := router.EnsureDefaultRoute(db); err != nil {
				log.Printf("seed routes for site %s: %v", site.ID, err)
			}
		}
		ctx := sites.WithContext(context.Background(), site)
		ext := s.chain.Extensions(site)
		if err := ext.Bootstrap(ctx); err != nil {
			log.Printf("extension bootstrap for %s: %v", site.ID, err)
		}
	}

	s.registerRoutes()
	return nil
}

func (s *Server) registerRoutes() {
	loginHandler := s.chain.Site(s.chain.Session(s.chain.CSRF(s.chain.Hooks(s.chain.ExtensionRequest(&admin.LoginHandler{Chain: s.chain})))))
	adminHandler := s.wrap(admin.NewHandler(s.chain, s.Activate, s.Reload))

	s.mux.Handle("/admin/login", loginHandler)
	s.mux.Handle("/admin", adminHandler)
	s.mux.Handle("/admin/", adminHandler)
	s.mux.Handle("/assets/", s.wrap(http.StripPrefix("/assets/", s.assetsHandler())))
	s.mux.Handle("/robots.txt", s.wrap(http.HandlerFunc(s.serveRobotsTXT)))
	s.mux.Handle("/sitemap.xml", s.wrap(http.HandlerFunc(s.serveSitemapXML)))

	apiHandler := api.NewHandler(s.chain)
	apiStack := s.chain.Site(s.chain.Hooks(s.chain.ExtensionRequest(apiHandler)))
	s.mux.Handle("/api/v1/", apiStack)
	s.mux.Handle("/api/v1", apiStack)

	// Completed installs should not expose the installer again.
	s.mux.HandleFunc("/install", func(w http.ResponseWriter, r *http.Request) {
		httpx.Redirect(w, r, "/admin/login")
	})
	s.mux.HandleFunc("/install/", func(w http.ResponseWriter, r *http.Request) {
		httpx.Redirect(w, r, "/admin/login")
	})

	s.mux.Handle("/", s.wrap(s.frontendHandler()))
}

func (s *Server) wrap(next http.Handler) http.Handler {
	return s.chain.Site(s.chain.Session(s.chain.CSRF(s.chain.Locale(s.chain.ContentLocale(s.chain.Hooks(s.chain.ExtensionRequest(next)))))))
}

func (s *Server) frontendHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		site, _ := sites.FromContext(r.Context())
		ext := s.chain.Extensions(site)

		var eng *templateengine.Engine
		renderCtx := func() context.Context {
			ctx := r.Context()
			if eng != nil {
				if hookCtx := eng.HookContext(); hookCtx != nil {
					return hookCtx
				}
			}
			return ctx
		}
		blockRenderer := func(name string) (template.HTML, error) {
			userCtx := extensionUserContext(r)
			ctx := renderCtx()
			var fragment blocks.FragmentRenderer
			if eng != nil {
				fragment = func(tmpl string, data map[string]any) (string, error) {
					var buf bytes.Buffer
					if err := eng.RenderFragment(&buf, tmpl, data); err != nil {
						return "", err
					}
					return buf.String(), nil
				}
			}
			html, err := blocks.RenderSpace(ctx, ext, name, r, userCtx, fragment)
			if err != nil {
				return "", err
			}
			if blocks.DebugSpacesActive(r.Context(), r) {
				html = blocks.WrapDebugSpace(name, html)
			}
			return template.HTML(html), nil
		}

		blockLen := func(name string) (int, error) {
			return blocks.CountForSpace(renderCtx(), ext, name)
		}

		sel, _ := themes.SelectionFromContext(r.Context())
		eng = templateengine.New(site, sel, blockRenderer, blockLen, templateengine.MergeFuncMaps(
			templateengine.CSRFFuncs(r),
			lang.FuncMap(middleware.LocaleFromContext(r.Context()), lang.TranslationPreviewActive(r)),
			template.FuncMap{
			"menu": func(name string) ([]map[string]any, error) {
				return router.MenuData(renderCtx(), name)
			},
			"siteName": func() string {
				return site.Name
			},
			"isOffline": func() bool {
				offline, err := settings.SiteOffline(r.Context())
				return err == nil && offline
			},
			"homeURL": func() string {
				return sites.DefaultRoutePath(r.Context())
			},
			"controllerURL": func(controller, action string) string {
				return routepath.Controller(r.Context(), controller, action)
			},
			"controllerURLFor": func(controller, action, suffix string) string {
				return routepath.ControllerWithSuffix(r.Context(), controller, action, suffix)
			},
			"isDefaultRoute": func() bool {
				return router.IsDefaultRouteContext(renderCtx())
			},
			"routeName": func() string {
				if route, ok := router.RouteFromContext(renderCtx()); ok {
					return route.Name
				}
				return ""
			},
			"routePath": func() string {
				if route, ok := router.RouteFromContext(renderCtx()); ok {
					return route.Path
				}
				return ""
			},
			"routeController": func() string {
				if route, ok := router.RouteFromContext(renderCtx()); ok {
					return route.Controller
				}
				return ""
			},
			"routeAction": func() string {
				if route, ok := router.RouteFromContext(renderCtx()); ok {
					return route.Action
				}
				return ""
			},
			"showRouteTitle": func() bool {
				if route, ok := router.MatchedRoute(renderCtx()); ok {
					return router.ShowTitleForRoute(route)
				}
				return true
			},
			"siteMetaDescription": func() string {
				desc, _ := settings.SiteMetaDescription(r.Context())
				return desc
			},
			"siteMetaKeywords": func() string {
				keywords, _ := settings.SiteMetaKeywords(r.Context())
				return keywords
			},
			"siteOGTitle": func() string {
				title, _ := settings.SiteOGTitle(r.Context())
				return title
			},
			"siteOGImage": func() string {
				image, _ := settings.SiteOGImage(r.Context())
				return image
			},
			"siteTwitterCard": func() string {
				card, _ := settings.SiteTwitterCard(r.Context())
				return card
			},
			"siteTwitterSite": func() string {
				site, _ := settings.SiteTwitterSite(r.Context())
				return site
			},
			"siteTwitterCreator": func() string {
				creator, _ := settings.SiteTwitterCreator(r.Context())
				return creator
			},
			"siteHeadExtra": func() template.HTML {
				raw, _ := settings.SiteHeadExtra(r.Context())
				return template.HTML(raw)
			},
			"year": func() int {
				return time.Now().Year()
			},
			"richText": func(src string) template.HTML {
				html, err := cms.RichTextToHTML(src)
				if err != nil {
					return template.HTML("")
				}
				return template.HTML(html)
			},
			"items": func(mode string, limit int) ([]models.Item, error) {
				viewerGroups, err := groups.ViewerGroupIDs(r.Context())
				if err != nil {
					return nil, err
				}
				opts := cms.ListOptions{Page: 1, Limit: limit}
				switch strings.TrimSpace(strings.ToLower(mode)) {
				case "featured":
					opts.Featured = true
				case "popular":
					return cms.PopularItems(r.Context(), viewerGroups, limit)
				}
				items, _, err := cms.ListItems(r.Context(), viewerGroups, opts)
				return items, err
			},
			"itemURL": func(slug string) string {
				return cms.ItemURLForContext(r.Context(), slug)
			},
			"categoryURL": func(slug string) string {
				return cms.CategoryURLForContext(r.Context(), slug)
			},
			"tagURL": func(slug string) string {
				return cms.TagURLForContext(r.Context(), slug)
			},
			"authorURL": func(key string) string {
				return cms.AuthorURLForContext(r.Context(), key)
			},
			"searchURL": func(query string) string {
				return cms.SearchURLForContext(r.Context(), query)
			},
			"featuredURL": func() string {
				return cms.FeaturedURLForContext(r.Context())
			},
			"categories": func() ([]models.Category, error) {
				return cms.CategoryTree(r.Context())
			},
			"tags": func() ([]models.Tag, error) {
				return cms.ListTags(r.Context())
			},
			"tagCloud": func(limit int) ([]cms.TagCount, error) {
				viewerGroups, err := groups.ViewerGroupIDs(r.Context())
				if err != nil {
					return nil, err
				}
				return cms.TagCloud(r.Context(), viewerGroups, limit)
			},
			"commentCount": func(itemID uint) (int64, error) {
				return cms.CommentCount(r.Context(), itemID)
			},
			"fieldOptions": func(raw string) []cms.FieldOption {
				return cms.ParseFieldConfig(raw).Options
			},
			"fieldValueContains": cms.FieldValueContains,
			"safeHTML": func(s string) template.HTML {
				return template.HTML(s)
			},
			"formatDateTimeLocal": func(t *time.Time) string {
				if t == nil || t.IsZero() {
					return ""
				}
				return t.Format("2006-01-02T15:04")
			},
		}))
		router.NewDispatcher(ext, eng).ServeHTTP(w, r)
	})
}

func extensionUserContext(r *http.Request) map[string]any {
	if userCtx, ok := router.UserContext(r.Context()); ok {
		return userCtx
	}
	svc, err := user.FromContext(r.Context())
	if err != nil {
		return map[string]any{"authenticated": false}
	}
	ctx, err := svc.Context(r.Context())
	if err != nil {
		return map[string]any{"authenticated": false}
	}
	return ctx
}

func (s *Server) serveThemeAsset(w http.ResponseWriter, r *http.Request) {
	if s.sites == nil {
		http.NotFound(w, r)
		return
	}
	site, err := s.sites.Resolve(r)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	r = r.WithContext(sites.WithContext(r.Context(), site))
	name := strings.TrimPrefix(r.URL.Path, "/theme/")
	name = strings.TrimPrefix(name, "/")
	if name == "" || strings.Contains(name, "..") {
		http.NotFound(w, r)
		return
	}
	theme, err := settings.FrontendTheme(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if !themes.IsBuiltinFrontend(theme) {
		assetsRoot := themes.AssetsDir(site.TemplateDir, theme)
		path := filepath.Join(assetsRoot, filepath.FromSlash(name))
		if info, err := os.Stat(path); err == nil && !info.IsDir() {
			http.ServeFile(w, r, path)
			return
		}
	}
	raw, _, err := templateengine.ThemeAsset(site.TemplateDir, theme, filepath.Base(name))
	if err != nil {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", detectContentType(name))
	w.Header().Set("Cache-Control", "public, max-age=3600")
	_, _ = w.Write(raw)
}

func (s *Server) serveAdminThemeAsset(w http.ResponseWriter, r *http.Request) {
	if s.sites == nil {
		http.NotFound(w, r)
		return
	}
	site, err := s.sites.Resolve(r)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	r = r.WithContext(sites.WithContext(r.Context(), site))
	name := strings.TrimPrefix(r.URL.Path, "/admin/assets/")
	name = strings.TrimPrefix(name, "/")
	if name == "" || strings.Contains(name, "..") {
		http.NotFound(w, r)
		return
	}
	theme, err := settings.AdminTheme(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if !themes.IsBuiltinAdmin(theme) {
		assetsRoot := themes.AssetsDir(site.TemplateDir, theme)
		path := filepath.Join(assetsRoot, filepath.FromSlash(name))
		if info, err := os.Stat(path); err == nil && !info.IsDir() {
			http.ServeFile(w, r, path)
			return
		}
	}
	raw, _, err := templateengine.AdminThemeAsset(site.TemplateDir, theme, filepath.Base(name))
	if err != nil {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", detectContentType(name))
	w.Header().Set("Cache-Control", "public, max-age=3600")
	_, _ = w.Write(raw)
}

func detectContentType(name string) string {
	switch strings.ToLower(filepath.Ext(name)) {
	case ".css":
		return "text/css; charset=utf-8"
	case ".js":
		return "application/javascript; charset=utf-8"
	case ".svg":
		return "image/svg+xml"
	case ".png":
		return "image/png"
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".webp":
		return "image/webp"
	case ".woff2":
		return "font/woff2"
	case ".woff":
		return "font/woff"
	default:
		return "application/octet-stream"
	}
}

func (s *Server) assetsHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		site, err := sites.FromContext(r.Context())
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		http.FileServer(http.Dir(site.AssetsDir)).ServeHTTP(w, r)
	})
}

// ListenAndServe starts the HTTP server.
func (s *Server) ListenAndServe(addr string) error {
	log.Printf("cannon listening on %s", addr)
	return http.ListenAndServe(addr, &muxDelegate{srv: s})
}
