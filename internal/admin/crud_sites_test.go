package admin

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rob121/cannon/internal/auth"
	"github.com/rob121/cannon/internal/config"
	"github.com/rob121/cannon/internal/database"
	"github.com/rob121/cannon/internal/groups"
	"github.com/rob121/cannon/internal/middleware"
	"github.com/rob121/cannon/internal/models"
	"github.com/rob121/cannon/internal/roles"
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
	if err := os.MkdirAll(filepath.Dir(cfg.Sites[0].Database.DSN), 0755); err != nil {
		t.Fatal(err)
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

func TestCopyAdminUsersToSiteCopiesLocalAdministrators(t *testing.T) {
	root := t.TempDir()
	source := testSiteConfig(root, "source-site", "http://source.test")
	target := testSiteConfig(root, "target-site", "http://target.test")

	prepareSiteForAdminCopy(t, &source)
	prepareSiteForAdminCopy(t, &target)

	ctx := sites.WithContext(context.Background(), &source)
	sourceDB, err := database.Get(source.ID)
	if err != nil {
		t.Fatal(err)
	}
	admin := models.User{
		GivenName:  "Ada",
		FamilyName: "Admin",
		Email:      "ada@example.test",
		Username:   "ada",
		Hash:       "copied-hash",
		Status:     models.StatusActive,
		Validated:  true,
	}
	if err := sourceDB.Create(&admin).Error; err != nil {
		t.Fatal(err)
	}
	if err := roles.AssignAdmin(ctx, admin.UserID); err != nil {
		t.Fatal(err)
	}

	if err := copyAdminUsersToSite(&source, &target); err != nil {
		t.Fatal(err)
	}

	targetDB, err := database.Get(target.ID)
	if err != nil {
		t.Fatal(err)
	}
	var copied models.User
	if err := targetDB.Preload("Groups").Where("username = ?", "ada").First(&copied).Error; err != nil {
		t.Fatal(err)
	}
	if copied.Hash != "copied-hash" || copied.Email != "ada@example.test" {
		t.Fatalf("copied user = %+v", copied)
	}
	if !hasGroup(copied.Groups, groups.AdministratorsGroupName) {
		t.Fatalf("copied user groups = %+v, want Administrators", copied.Groups)
	}
}

func prepareSiteForAdminCopy(t *testing.T, site *config.SiteConfig) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(site.Database.DSN), 0755); err != nil {
		t.Fatal(err)
	}
	if _, err := database.Connect(site); err != nil {
		t.Fatal(err)
	}
	if err := database.Migrate(site); err != nil {
		t.Fatal(err)
	}
	db, err := database.Get(site.ID)
	if err != nil {
		t.Fatal(err)
	}
	if err := auth.SeedAuthenticators(db); err != nil {
		t.Fatal(err)
	}
	if err := roles.EnsureDefaults(db); err != nil {
		t.Fatal(err)
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

func hasGroup(rows []models.Group, name string) bool {
	for _, row := range rows {
		if row.Name == name {
			return true
		}
	}
	return false
}
