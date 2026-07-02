package extension

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestMetaIncludesRouteHash(t *testing.T) {
	s := New(Info{Name: "demo-ext", Version: "1"})
	s.siteID = "site-a"
	req := httptest.NewRequest(http.MethodGet, "/meta", nil)
	rec := httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, req)

	var meta struct {
		RouteHash string `json:"route_hash"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&meta); err != nil {
		t.Fatal(err)
	}
	if meta.RouteHash != RouteHash("demo-ext", "site-a") {
		t.Fatalf("route_hash: got %q", meta.RouteHash)
	}
}
