package cache_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/rob121/cannon/internal/cache"
)

func TestBuildPageKey(t *testing.T) {
	if got := cache.BuildPageKey("en-US", "/blog", "page=2"); got != "en-US|/blog?page=2" {
		t.Fatalf("got %q", got)
	}
	if got := cache.BuildPageKey("", "", ""); got != "_|/" {
		t.Fatalf("got %q", got)
	}
}

func TestShouldCacheRequest(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/hello", nil)
	if !cache.ShouldCacheRequest(req, false) {
		t.Fatal("expected cacheable")
	}
	if cache.ShouldCacheRequest(req, true) {
		t.Fatal("authenticated should skip")
	}
	post := httptest.NewRequest(http.MethodPost, "/hello", nil)
	if cache.ShouldCacheRequest(post, false) {
		t.Fatal("POST should skip")
	}
	admin := httptest.NewRequest(http.MethodGet, "/admin/items", nil)
	if cache.ShouldCacheRequest(admin, false) {
		t.Fatal("admin should skip")
	}
	debug := httptest.NewRequest(http.MethodGet, "/hello?tp=1", nil)
	if cache.ShouldCacheRequest(debug, false) {
		t.Fatal("tp=1 should skip")
	}
}

func TestShouldCacheResponse(t *testing.T) {
	if !cache.ShouldCacheResponse(200, http.Header{}) {
		t.Fatal("expected ok")
	}
	if cache.ShouldCacheResponse(404, nil) {
		t.Fatal("non-200")
	}
	h := http.Header{}
	h.Set("Set-Cookie", "a=b")
	if cache.ShouldCacheResponse(200, h) {
		t.Fatal("set-cookie")
	}
}

func TestPageCacheTTLAndInvalidate(t *testing.T) {
	site := "test-page-cache"
	cache.InvalidateSitePages(site)
	key := cache.BuildPageKey("en-US", "/", "")
	cache.SetPage(site, key, 200, "text/html", []byte("<p>hi</p>"), time.Now().Add(time.Hour))
	ent, ok := cache.GetPage(site, key)
	if !ok || string(ent.Body) != "<p>hi</p>" {
		t.Fatalf("hit=%v body=%q", ok, ent.Body)
	}
	cache.SetPage(site, key+"-exp", 200, "text/html", []byte("x"), time.Now().Add(-time.Second))
	if _, ok := cache.GetPage(site, key+"-exp"); ok {
		t.Fatal("expired should miss")
	}
	cache.InvalidateSitePages(site)
	if _, ok := cache.GetPage(site, key); ok {
		t.Fatal("expected flush")
	}
}

func TestObjectCache(t *testing.T) {
	site := "test-obj-cache"
	cache.InvalidateSiteObjects(site)
	key := cache.ObjectKey("item", "42")
	cache.SetObject(site, key, "payload", time.Time{})
	v, ok := cache.GetObject(site, key)
	if !ok || v != "payload" {
		t.Fatalf("got %#v ok=%v", v, ok)
	}
	cache.InvalidateSitePublic(site)
	if _, ok := cache.GetObject(site, key); ok {
		t.Fatal("expected object flush")
	}
}
