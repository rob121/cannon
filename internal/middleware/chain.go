package middleware

import (
	"context"
	"io"
	"net/http"
	"strings"
	"sync"

	"github.com/rob121/cannon/internal/config"
	"github.com/rob121/cannon/internal/csrf"
	"github.com/rob121/cannon/internal/extensions"
	"github.com/rob121/cannon/internal/lang"
	"github.com/rob121/cannon/internal/router"
	"github.com/rob121/cannon/internal/session"
	"github.com/rob121/cannon/internal/sites"
	"github.com/rob121/cannon/internal/user"
)

type localeKey struct{}

// Chain builds common middleware for site requests.
type Chain struct {
	Sites      *sites.Manager
	extBySite  map[string]*extensions.Manager
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
		site, err := c.Sites.Resolve(r)
		if err != nil {
			http.Error(w, "site not found", http.StatusNotFound)
			return
		}
		ctx := sites.WithContext(r.Context(), site)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
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
		ctx := user.WithContext(r.Context(), svc)
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

func (c *Chain) ExtensionRequest(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		site, _ := sites.FromContext(r.Context())
		mgr := c.Extensions(site)
		svc, _ := user.FromContext(r.Context())
		userCtx, _ := svc.Context(r.Context())
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
