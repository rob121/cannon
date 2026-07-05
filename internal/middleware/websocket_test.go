package middleware

import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/rob121/cannon/internal/accesslog"
	"github.com/rob121/cannon/internal/config"
	"github.com/rob121/cannon/internal/sites"
)

func TestSiteMiddlewarePreservesHijackerForWebSocket(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "demo.sqlite")
	tmpDir := t.TempDir()
	cfg := &config.App{
		Session: config.SessionConfig{CookieName: "test_session", MaxAge: 3600},
		Sites: []config.SiteConfig{{
			ID:     "demo",
			Name:   "Demo",
			Host:   "http://127.0.0.1:8001",
			TmpDir: tmpDir,
			Database: config.DatabaseConfig{
				Type: "sqlite",
				DSN:  dbPath,
			},
		}},
	}
	mgr, err := sites.NewManager(cfg)
	if err != nil {
		t.Fatal(err)
	}
	chain := NewChain(mgr)

	var sawAccesslogWrapper bool
	handler := chain.Site(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, sawAccesslogWrapper = w.(*accesslog.ResponseWriter)
	}))

	req := httptest.NewRequest(http.MethodGet, "/connection/websocket", nil)
	req.Header.Set("Connection", "Upgrade")
	req.Header.Set("Upgrade", "websocket")
	req.Header.Set("Sec-WebSocket-Version", "13")
	req.Header.Set("Sec-WebSocket-Key", "dGhlIHNhbXBsZSBub25jZQ==")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)
	if sawAccesslogWrapper {
		t.Fatal("websocket upgrade should bypass accesslog ResponseWriter wrapper")
	}
}

func TestIsWebSocketUpgrade(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Connection", "keep-alive, Upgrade")
	req.Header.Set("Upgrade", "websocket")
	if !isWebSocketUpgrade(req) {
		t.Fatal("expected websocket upgrade detection")
	}
}
