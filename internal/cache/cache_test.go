package cache_test

import (
	"context"
	"testing"

	"github.com/rob121/cannon/internal/cache"
	"github.com/rob121/cannon/internal/config"
	"github.com/rob121/cannon/internal/database"
	"github.com/rob121/cannon/internal/models"
	"github.com/rob121/cannon/internal/settings"
	"github.com/rob121/cannon/internal/sites"
	"gorm.io/gorm"
)

func testSite(t *testing.T) (*config.SiteConfig, context.Context, *gorm.DB) {
	t.Helper()
	site := &config.SiteConfig{
		ID: t.Name(),
		Database: config.DatabaseConfig{
			Type: "sqlite",
			DSN:  t.TempDir() + "/site.db",
		},
	}
	if err := database.Migrate(site); err != nil {
		t.Fatal(err)
	}
	db, err := database.Get(site.ID)
	if err != nil {
		t.Fatal(err)
	}
	ctx := sites.WithContext(context.Background(), site)
	return site, ctx, db
}

func TestSettingsLoadReturnsCopy(t *testing.T) {
	cache.InvalidateAll()
	_, ctx, _ := testSite(t)
	cache.SetToggle(cache.SiteIDFromContext(ctx), true)

	store := settings.NewStore()
	if err := store.Save(ctx, settings.ScopeGlobal, settings.SectionGeneral, map[string]any{
		settings.ToggleKey(): true,
		"site_name":          "Original",
	}); err != nil {
		t.Fatal(err)
	}
	data, err := store.Load(ctx, settings.ScopeGlobal, settings.SectionGeneral)
	if err != nil {
		t.Fatal(err)
	}
	data["site_name"] = "Mutated"
	again, err := store.Load(ctx, settings.ScopeGlobal, settings.SectionGeneral)
	if err != nil {
		t.Fatal(err)
	}
	if again["site_name"] != "Original" {
		t.Fatalf("cache mutated by caller: %#v", again)
	}
}

func TestSettingsCacheRoundTrip(t *testing.T) {
	cache.InvalidateAll()
	_, ctx, _ := testSite(t)
	cache.SetToggle(cache.SiteIDFromContext(ctx), true)

	if err := settings.NewStore().Save(ctx, settings.ScopeGlobal, settings.SectionGeneral, map[string]any{
		settings.ToggleKey(): true,
		"site_name":          "Cached",
	}); err != nil {
		t.Fatal(err)
	}

	store := settings.NewStore()
	first, err := store.Load(ctx, settings.ScopeGlobal, settings.SectionGeneral)
	if err != nil {
		t.Fatal(err)
	}
	second, err := store.Load(ctx, settings.ScopeGlobal, settings.SectionGeneral)
	if err != nil {
		t.Fatal(err)
	}
	if first["site_name"] != "Cached" || second["site_name"] != "Cached" {
		t.Fatalf("settings=%#v", second)
	}

	if err := store.Save(ctx, settings.ScopeGlobal, settings.SectionGeneral, map[string]any{
		settings.ToggleKey(): true,
		"site_name":          "Updated",
	}); err != nil {
		t.Fatal(err)
	}
	third, err := store.Load(ctx, settings.ScopeGlobal, settings.SectionGeneral)
	if err != nil {
		t.Fatal(err)
	}
	if third["site_name"] != "Updated" {
		t.Fatalf("expected updated settings, got %#v", third)
	}
}

func TestEnabledRespectsToggle(t *testing.T) {
	cache.InvalidateAll()
	_, ctx, db := testSite(t)

	if cache.Enabled(ctx) {
		// default true when unset
	} else {
		t.Fatal("expected cache enabled by default")
	}

	if err := db.Where("scope = ? AND section = ?", "global", "general").
		Assign(models.Setting{Data: `{"memory_cache_enabled":false}`}).
		FirstOrCreate(&models.Setting{Scope: "global", Section: "general"}).Error; err != nil {
		t.Fatal(err)
	}
	cache.InvalidateAll()
	if cache.Enabled(ctx) {
		t.Fatal("expected cache disabled from settings")
	}
}

func TestActiveRoutesCache(t *testing.T) {
	cache.InvalidateAll()
	_, ctx, db := testSite(t)
	siteID := cache.SiteIDFromContext(ctx)
	cache.SetToggle(siteID, true)

	if err := db.Create(&models.Route{
		Name:   "Cache Probe",
		Path:   "/cache-probe",
		Type:   models.RouteTypeController,
		Status: models.StatusActive,
	}).Error; err != nil {
		t.Fatal(err)
	}

	routes, err := cache.ActiveRoutes(ctx, db)
	if err != nil || len(routes) == 0 {
		t.Fatalf("routes=%d err=%v", len(routes), err)
	}

	if err := db.Where("path = ?", "/cache-probe").Delete(&models.Route{}).Error; err != nil {
		t.Fatal(err)
	}
	cached, err := cache.ActiveRoutes(ctx, db)
	if err != nil || len(cached) == 0 {
		t.Fatalf("expected cached routes, got %d err=%v", len(cached), err)
	}

	cache.InvalidateRoutes(siteID)
	fresh, err := cache.ActiveRoutes(ctx, db)
	if err != nil {
		t.Fatal(err)
	}
	for _, route := range fresh {
		if route.Path == "/cache-probe" {
			t.Fatalf("expected probe route removed after invalidation: %+v", fresh)
		}
	}
}
