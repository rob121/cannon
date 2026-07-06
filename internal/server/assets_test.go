package server

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/rob121/cannon/internal/config"
)

func TestServeAdminThemeAssetDuringInstall(t *testing.T) {
	s := &Server{cfg: &config.App{InstallEnabled: true}}
	s.mux = http.NewServeMux()
	s.mux.HandleFunc("/admin/assets/", s.serveAdminThemeAsset)

	req := httptest.NewRequest(http.MethodGet, "/admin/assets/admin.css", nil)
	rec := httptest.NewRecorder()
	s.mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "text/css; charset=utf-8" {
		t.Fatalf("content-type = %q", ct)
	}
	if len(rec.Body.Bytes()) == 0 {
		t.Fatal("expected embedded admin.css body")
	}
}
