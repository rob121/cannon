package extension

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestMetaIncludesRouteHash(t *testing.T) {
	s := New(Info{
		Name:        "demo-ext",
		Title:       "Demo Extension",
		Description: "Example extension for tests.",
		Version:     "1",
	})
	s.siteID = "site-a"
	req := httptest.NewRequest(http.MethodGet, "/meta", nil)
	rec := httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, req)

	var meta struct {
		Title       string `json:"title"`
		Description string `json:"description"`
		RouteHash   string `json:"route_hash"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&meta); err != nil {
		t.Fatal(err)
	}
	if meta.Title != "Demo Extension" || meta.Description != "Example extension for tests." {
		t.Fatalf("meta: %+v", meta)
	}
	if meta.RouteHash != RouteHash("demo-ext", "site-a") {
		t.Fatalf("route_hash: got %q", meta.RouteHash)
	}
}
