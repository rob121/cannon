package admin

import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rob121/cannon/internal/config"
	"github.com/rob121/cannon/internal/middleware"
	"github.com/rob121/cannon/internal/sites"
)

func TestSaveSiteRequiresHost(t *testing.T) {
	root := t.TempDir()
	t.Setenv("CANNON_CONFIG", filepath.Join(root, "sites.json"))

	cfg := &config.App{DataRoot: filepath.Join(root, "data")}
	mgr, err := sites.NewManager(cfg)
	if err != nil {
		t.Fatal(err)
	}
	h := &Handler{chain: middleware.NewChain(mgr)}

	body := strings.NewReader("name=Second+Site&host=&database_type=sqlite")
	req := httptest.NewRequest(http.MethodPost, "/admin/sites/new", body)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	var row config.SiteConfig
	if err := h.saveSite(cfg, &row, true, req); err == nil {
		t.Fatal("expected blank host to fail")
	}

	if len(cfg.Sites) != 0 {
		t.Fatalf("sites len = %d, want 0", len(cfg.Sites))
	}
}

func TestSaveSiteRejectsDuplicateHost(t *testing.T) {
	root := t.TempDir()
	t.Setenv("CANNON_CONFIG", filepath.Join(root, "sites.json"))

	cfg := &config.App{
		DataRoot: filepath.Join(root, "data"),
		Sites: []config.SiteConfig{{
			ID:          "default",
			Name:        "Default",
			Host:        "http://localhost:8001",
			TemplateDir: filepath.Join(root, "data", "default", "templates"),
			AssetsDir:   filepath.Join(root, "data", "default", "assets"),
			TmpDir:      filepath.Join(root, "data", "default", "tmp"),
			LanguageDir: filepath.Join(root, "data", "default", "language"),
			Database: config.DatabaseConfig{
				Type: "sqlite",
				DSN:  filepath.Join(root, "data", "default", "default.sqlite"),
			},
		}},
	}
	mgr, err := sites.NewManager(cfg)
	if err != nil {
		t.Fatal(err)
	}
	h := &Handler{chain: middleware.NewChain(mgr)}

	body := strings.NewReader("name=Second+Site&host=http%3A%2F%2F127.0.0.1%3A8001&database_type=sqlite")
	req := httptest.NewRequest(http.MethodPost, "/admin/sites/new", body)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	var row config.SiteConfig
	if err := h.saveSite(cfg, &row, true, req); err == nil {
		t.Fatal("expected duplicate localhost host to fail")
	}
	if len(cfg.Sites) != 1 {
		t.Fatalf("sites len = %d, want 1", len(cfg.Sites))
	}
}
