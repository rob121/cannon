package extension_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/rob121/cannon/extension"
)

func TestPageResolve(t *testing.T) {
	s := extension.New(extension.Info{Name: "test", Version: "1"})
	s.RegisterPage(extension.PageDefinition{ID: "index", Title: "Index"}, func(item string, req extension.WireRequest) extension.WireResponse {
		return extension.HTML(200, "index")
	})
	s.RegisterPage(extension.PageDefinition{ID: "detail", Title: "Detail"}, func(item string, req extension.WireRequest) extension.WireResponse {
		return extension.HTML(200, "detail:"+extension.WirePathSuffix(req))
	})
	s.OnPageResolve(func(req extension.WireRequest) extension.PageResolveResponse {
		if extension.WirePathSuffix(req) == "" {
			return extension.PageResolveResponse{PageItem: "index"}
		}
		return extension.PageResolveResponse{PageItem: "detail"}
	})

	req := httptest.NewRequest(http.MethodGet, "/capabilities", nil)
	rec := httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, req)
	var caps struct {
		Capabilities map[string]string `json:"capabilities"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&caps); err != nil {
		t.Fatal(err)
	}
	if caps.Capabilities["page_resolve"] != "/page/resolve" {
		t.Fatalf("page_resolve cap: %+v", caps.Capabilities)
	}

	body, _ := json.Marshal(extension.WireRequest{
		Method:     http.MethodGet,
		URL:        "/jobs",
		RoutePath:  "/jobs/*",
		PathSuffix: "",
		SiteID:     "site-1",
	})
	req = httptest.NewRequest(http.MethodPost, "/page/resolve", bytes.NewReader(body))
	rec = httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, req)
	var resolved extension.PageResolveResponse
	if err := json.NewDecoder(rec.Body).Decode(&resolved); err != nil {
		t.Fatal(err)
	}
	if resolved.PageItem != "index" {
		t.Fatalf("resolve index: %+v", resolved)
	}

	body, _ = json.Marshal(extension.WireRequest{
		Method:     http.MethodGet,
		URL:        "/jobs/acme",
		RoutePath:  "/jobs/*",
		PathSuffix: "acme",
		SiteID:     "site-1",
	})
	req = httptest.NewRequest(http.MethodPost, "/page/resolve", bytes.NewReader(body))
	rec = httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, req)
	if err := json.NewDecoder(rec.Body).Decode(&resolved); err != nil {
		t.Fatal(err)
	}
	if resolved.PageItem != "detail" {
		t.Fatalf("resolve detail: %+v", resolved)
	}
}
