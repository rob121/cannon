package admin

import (
	"context"
	"net/http/httptest"
	"testing"

	"github.com/rob121/cannon/internal/config"
	"github.com/rob121/cannon/internal/database"
	"github.com/rob121/cannon/internal/settings"
	"github.com/rob121/cannon/internal/sites"
)

func TestThemeTypeLabel(t *testing.T) {
	if got := themeTypeLabel("frontend"); got != "Frontend" {
		t.Fatalf("got %q", got)
	}
	if got := themeTypeLabel("backend"); got != "Admin" {
		t.Fatalf("got %q", got)
	}
	if got := themeTypeLabel("full"); got != "Full" {
		t.Fatalf("got %q", got)
	}
}

func TestRoleDisplayName(t *testing.T) {
	if got := RoleDisplayName("administrator"); got != "Administrator" {
		t.Fatalf("got %q", got)
	}
	if got := RoleDisplayName("content_editor"); got != "Content Editor" {
		t.Fatalf("got %q", got)
	}
}

func TestAdminSiteDisplayNamePrefersGeneralSetting(t *testing.T) {
	site := &config.SiteConfig{
		ID:   "admin-brand-display-name-test",
		Name: "Config Site",
		Database: config.DatabaseConfig{
			Type: "sqlite",
			DSN:  t.TempDir() + "/site.db",
		},
	}
	if err := database.Migrate(site); err != nil {
		t.Fatal(err)
	}
	ctx := sites.WithContext(context.Background(), site)
	if err := settings.Save(ctx, settings.NewStore(), settings.ScopeGlobal, settings.SectionGeneral, map[string]any{
		"site_name": "Display Site",
	}); err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest("GET", "/admin", nil).WithContext(ctx)
	if got := adminSiteDisplayName(req, site); got != "Display Site" {
		t.Fatalf("got %q", got)
	}
}

func TestAdminSiteDisplayNameFallsBackToSiteConfig(t *testing.T) {
	site := &config.SiteConfig{ID: "admin-brand-fallback-test", Name: "Config Site"}
	req := httptest.NewRequest("GET", "/admin", nil).WithContext(sites.WithContext(context.Background(), site))
	if got := adminSiteDisplayName(req, site); got != "Config Site" {
		t.Fatalf("got %q", got)
	}
}
