package middleware

import (
	"context"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/rob121/cannon/internal/accesslog"
	"github.com/rob121/cannon/internal/applog"
	"github.com/rob121/cannon/internal/appupdate"
	"github.com/rob121/cannon/internal/config"
	"github.com/rob121/cannon/internal/content"
	"github.com/rob121/cannon/internal/csrf"
	"github.com/rob121/cannon/internal/extensions"
	"github.com/rob121/cannon/internal/httpreq"
	"github.com/rob121/cannon/internal/lang"
	"github.com/rob121/cannon/internal/router"
	"github.com/rob121/cannon/internal/security"
	"github.com/rob121/cannon/internal/session"
	"github.com/rob121/cannon/internal/settings"
	"github.com/rob121/cannon/internal/sites"
	"github.com/rob121/cannon/internal/user"
)

type localeKey struct{}

// Chain builds common middleware for site requests.
type Chain struct {
	Sites      *sites.Manager
	extBySite  map[string]*extensions.Manager
	appUpdates *appupdate.Manager
	sessionMap map[string]*session.Store
	langMap    map[string]*lang.Manager
	mu         sync.RWMutex
}

func NewChain(m *sites.Manager) *Chain {
	return &Chain{
		Sites:      m,
		extBySite:  map[string]*extensions.Manager{},
		sessionMap: map[string]*session.Store{},
		langMap:    map[string]*lang.Manager{},
	}
}

func (c *Chain) Extensions(site *config.SiteConfig) *extensions.Manager {
	c.mu.RLock()
	if mgr, ok := c.extBySite[site.ID]; ok {
		c.mu.RUnlock()
		return mgr
	}
	c.mu.RUnlock()

	c.mu.Lock()
	defer c.mu.Unlock()
	if mgr, ok := c.extBySite[site.ID]; ok {
		return mgr
	}
	mgr := extensions.NewManager(c.Sites.Config(), site)
	c.extBySite[site.ID] = mgr
	return mgr
}

func (c *Chain) AppUpdates() *appupdate.Manager {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.appUpdates == nil {
		c.appUpdates = appupdate.New(c.Sites.Config())
		c.appUpdates.StartChecker()
	}
	return c.appUpdates
}

func (c *Chain) SessionStore(site *config.SiteConfig) (*session.Store, error) {
	c.mu.RLock()
	if store, ok := c.sessionMap[site.ID]; ok {
		c.mu.RUnlock()
		return store, nil
	}
	c.mu.RUnlock()

	store, err := session.NewStore(site, c.Sites.Config())
	if err != nil {
		return nil, err
	}
	c.mu.Lock()
	c.sessionMap[site.ID] = store
	c.mu.Unlock()
	return store, nil
}

func (c *Chain) Lang(site *config.SiteConfig, locale string) (*lang.Manager, error) {
	key := site.ID + ":" + locale
	c.mu.RLock()
	if mgr, ok := c.langMap[key]; ok {
		c.mu.RUnlock()
		return mgr, nil
	}
	c.mu.RUnlock()

	mgr, err := lang.NewManager(site, locale)
	if err != nil {
		return nil, err
	}
	c.mu.Lock()
	c.langMap[key] = mgr
	c.mu.Unlock()
	return mgr, nil
}

// InvalidateLang drops cached language managers for a site after admin edits.
func (c *Chain) InvalidateLang(site *config.SiteConfig) {
	if site == nil {
		return
	}
	prefix := site.ID + ":"
	c.mu.Lock()
	for key := range c.langMap {
		if strings.HasPrefix(key, prefix) {
			delete(c.langMap, key)
		}
	}
	c.mu.Unlock()
}

func (c *Chain) Site(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		site, err := c.Sites.Resolve(r)
		if err != nil {
			writeNoSitesConfigured(w)
			return
		}
		ctx := sites.WithContext(r.Context(), site)
		if level, err := settings.LogLevel(ctx); err == nil {
			applog.SetLevel(applog.ParseLevel(level))
		}

		// WebSocket upgrades must reach the origin ResponseWriter (Hijacker).
		if isWebSocketUpgrade(r) {
			next.ServeHTTP(w, r.WithContext(ctx))
			return
		}

		rw := &accesslog.ResponseWriter{ResponseWriter: w, Status: http.StatusOK}
		next.ServeHTTP(rw, r.WithContext(ctx))
		if !rw.Hijacked {
			accesslog.Write(site, r, rw.Status, rw.Bytes, time.Since(start))
		}
	})
}

func writeNoSitesConfigured(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusServiceUnavailable)
	_, _ = io.WriteString(w, `<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>No Sites Configured</title>
  <style>
    :root { color-scheme: light dark; }
    body {
      margin: 0;
      min-height: 100vh;
      display: grid;
      place-items: center;
      font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif;
      background: #f6f7f9;
      color: #172033;
    }
    main {
      width: min(34rem, calc(100vw - 2rem));
      padding: 2rem;
      border: 1px solid #d9dee8;
      border-radius: 0.5rem;
      background: #ffffff;
      box-shadow: 0 12px 32px rgba(15, 23, 42, 0.08);
    }
    h1 {
      margin: 0 0 0.75rem;
      font-size: 1.25rem;
      line-height: 1.25;
    }
    p {
      margin: 0;
      color: #526070;
      line-height: 1.55;
    }
    code {
      padding: 0.125rem 0.25rem;
      border-radius: 0.25rem;
      background: #eef1f5;
      color: #172033;
    }
    @media (prefers-color-scheme: dark) {
      body { background: #0f172a; color: #e5e7eb; }
      main { background: #111827; border-color: #334155; box-shadow: none; }
      p { color: #cbd5e1; }
      code { background: #1f2937; color: #f8fafc; }
    }
  </style>
</head>
<body>
  <main>
    <h1>No Sites Configured</h1>
    <p>Cannon could not find any sites in <code>sites.json</code>. Add at least one site entry, then restart the server.</p>
  </main>
</body>
</html>`)
}

func isWebSocketUpgrade(r *http.Request) bool {
	return strings.Contains(strings.ToLower(r.Header.Get("Connection")), "upgrade") &&
		strings.EqualFold(r.Header.Get("Upgrade"), "websocket")
}

func (c *Chain) Session(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		site, err := sites.FromContext(r.Context())
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		store, err := c.SessionStore(site)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		cookie, _ := r.Cookie(store.CookieName())
		sessionID := ""
		if cookie != nil {
			sessionID = cookie.Value
		}
		if sessionID == "" {
			sessionID, _ = store.Create()
			http.SetCookie(w, &http.Cookie{
				Name:     store.CookieName(),
				Value:    sessionID,
				Path:     "/",
				MaxAge:   store.MaxAge(),
				HttpOnly: true,
				SameSite: http.SameSiteLaxMode,
			})
		}

		svc, err := user.NewService(store, sessionID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		// Drop the Goth cookie left by older builds once, when the browser
		// still sends it. Clearing it on every response sets Set-Cookie and
		// keeps the anonymous page cache from storing the page.
		if _, err := r.Cookie("_gothic_session"); err == nil {
			http.SetCookie(w, &http.Cookie{
				Name:     "_gothic_session",
				Value:    "",
				Path:     "/",
				MaxAge:   -1,
				HttpOnly: true,
				SameSite: http.SameSiteLaxMode,
			})
		}
		ctx := user.WithContext(r.Context(), svc)
		if userID, ok := svc.CurrentID(); ok {
			if loaded, err := security.PreloadForUser(ctx, userID); err == nil {
				ctx = loaded
			}
		}
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (c *Chain) CSRF(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		svc, err := user.FromContext(r.Context())
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if csrf.IsMutating(r.Method) {
			if err := svc.ValidateCSRF(r); err != nil {
				http.Error(w, "invalid csrf token", http.StatusForbidden)
				return
			}
		} else {
			if _, err := svc.EnsureCSRFToken(); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}

func (c *Chain) Locale(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		site, _ := sites.FromContext(r.Context())
		cookie, _ := r.Cookie("cannon_locale")
		localeCookie := ""
		if cookie != nil {
			localeCookie = cookie.Value
		}
		locale := lang.ResolveLocale(localeCookie, r.Header.Get("Accept-Language"))
		mgr, err := c.Lang(site, locale)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		ctx := context.WithValue(r.Context(), localeKey{}, mgr)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (c *Chain) ContentLocale(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		uiLocale := ""
		if mgr := LocaleFromContext(r.Context()); mgr != nil {
			uiLocale = mgr.Locale()
		}
		cookie, _ := r.Cookie("cannon_locale")
		cookieVal := ""
		if cookie != nil {
			cookieVal = cookie.Value
		}

		if strings.HasPrefix(path, "/admin") {
			_, defaultLocale, _, _ := content.LocaleConfig(r.Context())
			ctx := content.WithLocale(r.Context(), defaultLocale)
			next.ServeHTTP(w, r.WithContext(ctx))
			return
		}

		locale, stripped := content.ResolveContentLocale(
			r.Context(),
			path,
			uiLocale,
			cookieVal,
			r.Header.Get("Accept-Language"),
		)
		if stripped != path {
			cloned := r.Clone(r.Context())
			cloned.URL.Path = stripped
			r = cloned
		}
		ctx := content.WithLocale(r.Context(), locale)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (c *Chain) ExtensionRequest(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		site, _ := sites.FromContext(r.Context())
		mgr := c.Extensions(site)
		var userCtx map[string]any
		if svc, err := user.FromContext(r.Context()); err == nil && svc != nil {
			userCtx, _ = svc.Context(r.Context())
		}
		if userCtx == nil {
			userCtx = map[string]any{"authenticated": false}
		}
		updated, resp, stop, err := mgr.HandleRequest(r.Context(), r, userCtx)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if stop && resp != nil {
			for k, vals := range resp.Header {
				for _, v := range vals {
					w.Header().Add(k, v)
				}
			}
			w.WriteHeader(resp.StatusCode)
			_, _ = ioCopy(w, resp.Body)
			return
		}
		ctx := router.WithUserContext(r.Context(), userCtx)
		ctx = extensions.WithContext(ctx, mgr)
		ctx = httpreq.WithContext(ctx, updated)
		next.ServeHTTP(w, updated.WithContext(ctx))
	})
}

func LocaleFromContext(ctx context.Context) *lang.Manager {
	mgr, _ := ctx.Value(localeKey{}).(*lang.Manager)
	return mgr
}

func ioCopy(w http.ResponseWriter, r io.Reader) (int64, error) {
	buf := make([]byte, 32*1024)
	var written int64
	for {
		n, err := r.Read(buf)
		if n > 0 {
			wn, werr := w.Write(buf[:n])
			written += int64(wn)
			if werr != nil {
				return written, werr
			}
		}
		if err != nil {
			if err.Error() == "EOF" {
				return written, nil
			}
			return written, err
		}
	}
}
