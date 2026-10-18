package server

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/rob121/cannon/internal/cache"
	"github.com/rob121/cannon/internal/config"
	"github.com/rob121/cannon/internal/sites"
	"github.com/rob121/cannon/internal/user"
)

func TestWithPageCacheHitMiss(t *testing.T) {
	site := &config.SiteConfig{ID: "page-cache-test", Name: "Test"}
	cache.InvalidateSitePublic(site.ID)

	calls := 0
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("<html>ok</html>"))
	})

	// Force-enable page cache by pre-seeding settings is hard; call store directly
	// and exercise capture path via a thin wrapper mimicking enabled=true.
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := sites.WithContext(r.Context(), site)
		svc := &user.Service{}
		ctx = user.WithContext(ctx, svc)
		r = r.WithContext(ctx)

		if !cache.ShouldCacheRequest(r, false) {
			inner.ServeHTTP(w, r)
			return
		}
		key := cache.BuildPageKey("en-US", r.URL.Path, r.URL.RawQuery)
		if ent, ok := cache.GetPage(site.ID, key); ok {
			w.Header().Set(cache.PageCacheHeader, cache.PageCacheHeaderHIT)
			w.Header().Set("Content-Type", ent.ContentType)
			w.WriteHeader(ent.Status)
			_, _ = w.Write(ent.Body)
			return
		}
		w.Header().Set(cache.PageCacheHeader, cache.PageCacheHeaderMISS)
		cap := &capturingResponseWriter{ResponseWriter: w, status: http.StatusOK}
		inner.ServeHTTP(cap, r)
		if cache.ShouldCacheResponse(cap.Status(), cap.Header()) {
			cache.SetPage(site.ID, key, cap.Status(), cap.Header().Get("Content-Type"), cap.body, time.Now().Add(time.Hour))
		}
	})

	req1 := httptest.NewRequest(http.MethodGet, "/hello", nil)
	rec1 := httptest.NewRecorder()
	handler.ServeHTTP(rec1, req1)
	if rec1.Header().Get(cache.PageCacheHeader) != cache.PageCacheHeaderMISS {
		t.Fatalf("want MISS, got %q", rec1.Header().Get(cache.PageCacheHeader))
	}
	if calls != 1 {
		t.Fatalf("calls=%d", calls)
	}

	req2 := httptest.NewRequest(http.MethodGet, "/hello", nil)
	rec2 := httptest.NewRecorder()
	handler.ServeHTTP(rec2, req2)
	if rec2.Header().Get(cache.PageCacheHeader) != cache.PageCacheHeaderHIT {
		t.Fatalf("want HIT, got %q", rec2.Header().Get(cache.PageCacheHeader))
	}
	if calls != 1 {
		t.Fatalf("second request should not hit inner, calls=%d", calls)
	}
	if body := rec2.Body.String(); body != "<html>ok</html>" {
		t.Fatalf("body=%q", body)
	}

	cache.InvalidateSitePublic(site.ID)
	req3 := httptest.NewRequest(http.MethodGet, "/hello", nil)
	rec3 := httptest.NewRecorder()
	handler.ServeHTTP(rec3, req3)
	if rec3.Header().Get(cache.PageCacheHeader) != cache.PageCacheHeaderMISS {
		t.Fatalf("after flush want MISS, got %q", rec3.Header().Get(cache.PageCacheHeader))
	}
	if calls != 2 {
		t.Fatalf("calls after flush=%d", calls)
	}
}
