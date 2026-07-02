package server

import (
	"context"
	"html/template"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/rob121/cannon/internal/admin"
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
	"github.com/rob121/cannon/internal/middleware"
	"github.com/rob121/cannon/internal/models"
	"github.com/rob121/cannon/internal/roles"
	"github.com/rob121/cannon/internal/router"
	"github.com/rob121/cannon/internal/sites"
	"github.com/rob121/cannon/internal/templateengine"
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
	s.mux.HandleFunc("/admin/assets/admin.css", s.serveAdminCSS)
	s.mux.HandleFunc("/admin/assets/admin.js", s.serveAdminJS)
	s.mux.HandleFunc("/admin/assets/cannon-icon.svg", s.serveAdminIcon)
	s.mux.HandleFunc("/theme/site.css", s.serveSiteCSS)
	s.mux.HandleFunc("/theme/cannon-icon.svg", s.serveSiteIcon)

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
	loginHandler := s.chain.Site(s.chain.Session(s.chain.CSRF(s.chain.Hooks(&admin.LoginHandler{Chain: s.chain}))))
	adminHandler := s.wrap(admin.NewHandler(s.chain, s.Activate, s.Reload))

	s.mux.Handle("/admin/login", loginHandler)
	s.mux.Handle("/admin", adminHandler)
	s.mux.Handle("/admin/", adminHandler)
	s.mux.Handle("/assets/", s.wrap(http.StripPrefix("/assets/", s.assetsHandler())))
	s.mux.Handle("/robots.txt", s.wrap(http.HandlerFunc(s.serveRobotsTXT)))

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
	return s.chain.Site(s.chain.Session(s.chain.CSRF(s.chain.Locale(s.chain.Hooks(s.chain.ExtensionRequest(next))))))
}

func (s *Server) frontendHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		site, _ := sites.FromContext(r.Context())
		ext := s.chain.Extensions(site)

		blockRenderer := func(name string) (template.HTML, error) {
			userCtx := extensionUserContext(r)
			html, err := blocks.RenderSpace(r.Context(), ext, name, r, userCtx)
			if err != nil {
				return "", err
			}
			if blocks.DebugSpacesActive(r.Context(), r) {
				html = blocks.WrapDebugSpace(name, html)
			}
			return template.HTML(html), nil
		}

		blockLen := func(name string) (int, error) {
			return blocks.CountForSpace(r.Context(), ext, name)
		}

		eng := templateengine.New(site, blockRenderer, blockLen, templateengine.MergeFuncMaps(
			templateengine.CSRFFuncs(r),
			template.FuncMap{
			"menu": func(name string) ([]map[string]string, error) {
				return router.MenuData(r.Context(), name)
			},
			"siteName": func() string {
				return site.Name
			},
			"year": func() int {
				return time.Now().Year()
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
			"itemURL": cms.ItemURL,
			"categoryURL": cms.CategoryURL,
			"tagURL": cms.TagURL,
			"authorURL": cms.AuthorURL,
			"searchURL": cms.SearchURL,
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

func (s *Server) serveAdminCSS(w http.ResponseWriter, r *http.Request) {
	s.serveAdminAsset(w, r, "admin.css", "text/css; charset=utf-8")
}

func (s *Server) serveAdminJS(w http.ResponseWriter, r *http.Request) {
	s.serveAdminAsset(w, r, "admin.js", "application/javascript; charset=utf-8")
}

func (s *Server) serveAdminIcon(w http.ResponseWriter, r *http.Request) {
	s.serveAdminAsset(w, r, "cannon-icon.svg", "image/svg+xml")
}

func (s *Server) serveAdminAsset(w http.ResponseWriter, r *http.Request, name, contentType string) {
	raw, err := templateengine.AdminAsset(name)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Cache-Control", "public, max-age=3600")
	_, _ = w.Write(raw)
}

func (s *Server) serveSiteCSS(w http.ResponseWriter, r *http.Request) {
	s.serveSiteAsset(w, r, "site.css", "text/css; charset=utf-8")
}

func (s *Server) serveSiteIcon(w http.ResponseWriter, r *http.Request) {
	s.serveSiteAsset(w, r, "cannon-icon.svg", "image/svg+xml")
}

func (s *Server) serveSiteAsset(w http.ResponseWriter, r *http.Request, name, contentType string) {
	raw, err := templateengine.SiteAsset(name)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Cache-Control", "public, max-age=3600")
	_, _ = w.Write(raw)
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
