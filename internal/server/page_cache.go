package server

import (
	"net/http"
	"time"

	"github.com/rob121/cannon/internal/cache"
	cms "github.com/rob121/cannon/internal/content"
	"github.com/rob121/cannon/internal/settings"
	"github.com/rob121/cannon/internal/sites"
	"github.com/rob121/cannon/internal/user"
)

type capturingResponseWriter struct {
	http.ResponseWriter
	status      int
	body        []byte
	wroteHeader bool
}

func (c *capturingResponseWriter) WriteHeader(code int) {
	if c.wroteHeader {
		return
	}
	c.wroteHeader = true
	c.status = code
	c.ResponseWriter.WriteHeader(code)
}

func (c *capturingResponseWriter) Write(p []byte) (int, error) {
	if !c.wroteHeader {
		c.WriteHeader(http.StatusOK)
	}
	c.body = append(c.body, p...)
	return c.ResponseWriter.Write(p)
}

func (c *capturingResponseWriter) Status() int {
	if c.status == 0 {
		return http.StatusOK
	}
	return c.status
}

func withPageCache(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		enabled, err := settings.PageCacheEnabled(r.Context())
		if err != nil || !enabled {
			next.ServeHTTP(w, r)
			return
		}
		authenticated := false
		if svc, err := user.FromContext(r.Context()); err == nil {
			_, authenticated = svc.CurrentID()
		}
		if !cache.ShouldCacheRequest(r, authenticated) {
			next.ServeHTTP(w, r)
			return
		}
		site, err := sites.FromContext(r.Context())
		if err != nil || site == nil || site.ID == "" {
			next.ServeHTTP(w, r)
			return
		}
		locale := cms.LocaleFromContext(r.Context())
		key := cache.BuildPageKey(locale, r.URL.Path, r.URL.RawQuery)
		if ent, ok := cache.GetPage(site.ID, key); ok {
			w.Header().Set(cache.PageCacheHeader, cache.PageCacheHeaderHIT)
			if ent.ContentType != "" {
				w.Header().Set("Content-Type", ent.ContentType)
			}
			w.WriteHeader(ent.Status)
			_, _ = w.Write(ent.Body)
			return
		}
		w.Header().Set(cache.PageCacheHeader, cache.PageCacheHeaderMISS)
		cap := &capturingResponseWriter{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(cap, r)
		if !cache.ShouldCacheResponse(cap.Status(), cap.Header()) {
			return
		}
		ttl, err := settings.PageCacheTTL(r.Context())
		if err != nil || ttl <= 0 {
			ttl = 300 * time.Second
		}
		cache.SetPage(site.ID, key, cap.Status(), cap.Header().Get("Content-Type"), cap.body, time.Now().Add(ttl))
	})
}
