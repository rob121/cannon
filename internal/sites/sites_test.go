package sites

import (
	"path/filepath"
	"testing"

	"github.com/rob121/cannon/internal/config"
)

func TestNewManagerRejectsDuplicateHostMappings(t *testing.T) {
	root := t.TempDir()
	cfg := &config.App{
		DataRoot: root,
		Sites: []config.SiteConfig{
			testSiteConfig(root, "default", "http://localhost:8001"),
			testSiteConfig(root, "second", "http://127.0.0.1:9000"),
		},
	}

	if _, err := NewManager(cfg); err == nil {
		t.Fatal("expected duplicate localhost mapping to fail")
	}
}

func testSiteConfig(root, id, host string) config.SiteConfig {
	siteDir := filepath.Join(root, id)
	return config.SiteConfig{
		ID:          id,
		Name:        id,
		Host:        host,
		TemplateDir: filepath.Join(siteDir, "templates"),
		AssetsDir:   filepath.Join(siteDir, "assets"),
		TmpDir:      filepath.Join(siteDir, "tmp"),
		LanguageDir: filepath.Join(siteDir, "language"),
		Database: config.DatabaseConfig{
			Type: "sqlite",
			DSN:  filepath.Join(siteDir, id+".sqlite"),
		},
	}
}
