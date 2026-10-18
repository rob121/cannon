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

func TestActiveMenuItemsCache(t *testing.T) {
	cache.InvalidateAll()
	_, ctx, db := testSite(t)
	siteID := cache.SiteIDFromContext(ctx)
	cache.SetToggle(siteID, true)

	menu := models.Menu{MenuName: "main", Status: models.StatusActive}
	if err := db.Create(&menu).Error; err != nil {
		t.Fatal(err)
	}
	route := models.Route{Name: "Home", Path: "/", Type: models.RouteTypeController, Status: models.StatusActive}
	if err := db.Create(&route).Error; err != nil {
		t.Fatal(err)
	}
	item := models.MenuItem{
		MenuID: menu.MenuID, Name: "Home", RouteID: &route.RouteID,
		Status: models.StatusActive, Sort: 1,
	}
	if err := db.Create(&item).Error; err != nil {
		t.Fatal(err)
	}

	first, err := cache.ActiveMenuItems(ctx, db, "main")
	if err != nil || len(first) != 1 {
		t.Fatalf("first=%d err=%v", len(first), err)
	}
	if err := db.Where("menu_id = ?", menu.MenuID).Delete(&models.MenuItem{}).Error; err != nil {
		t.Fatal(err)
	}
	cached, err := cache.ActiveMenuItems(ctx, db, "main")
	if err != nil || len(cached) != 1 {
		t.Fatalf("expected cached menu items, got %d err=%v", len(cached), err)
	}
	cache.InvalidateMenus(siteID)
	fresh, err := cache.ActiveMenuItems(ctx, db, "main")
	if err != nil {
		t.Fatal(err)
	}
	if len(fresh) != 0 {
		t.Fatalf("expected empty after invalidate, got %d", len(fresh))
	}
}

func TestViewerGroupsLRUEvicts(t *testing.T) {
	cache.InvalidateAll()
	prev := cache.MaxViewerGroupsPerSite
	cache.MaxViewerGroupsPerSite = 2
	defer func() {
		cache.MaxViewerGroupsPerSite = prev
		cache.InvalidateAll()
	}()

	_, ctx, db := testSite(t)
	cache.SetToggle(cache.SiteIDFromContext(ctx), true)

	if err := db.Where("name = ?", "public").
		FirstOrCreate(&models.Group{Name: "public", Kind: models.GroupKindFrontend, Status: models.StatusActive}).Error; err != nil {
		t.Fatal(err)
	}

	loadCalls := map[uint]int{}
	load := func(_ *gorm.DB, userID uint) ([]uint, error) {
		loadCalls[userID]++
		return []uint{userID}, nil
	}

	for _, uid := range []uint{1, 2, 3} {
		if _, err := cache.ViewerGroupIDs(ctx, db, uid, load); err != nil {
			t.Fatal(err)
		}
	}
	// User 1 should have been evicted; next access reloads.
	if _, err := cache.ViewerGroupIDs(ctx, db, 1, load); err != nil {
		t.Fatal(err)
	}
	if loadCalls[1] < 2 {
		t.Fatalf("expected user 1 reloaded after LRU eviction, calls=%v", loadCalls)
	}
	if loadCalls[2] != 1 || loadCalls[3] != 1 {
		t.Fatalf("unexpected load counts: %v", loadCalls)
	}
}

func TestBlockSpacesLRUEvicts(t *testing.T) {
	cache.InvalidateAll()
	prev := cache.MaxBlockSpacesPerSite
	cache.MaxBlockSpacesPerSite = 2
	defer func() {
		cache.MaxBlockSpacesPerSite = prev
		cache.InvalidateAll()
	}()

	_, ctx, db := testSite(t)
	cache.SetToggle(cache.SiteIDFromContext(ctx), true)

	for i, space := range []string{"a", "b", "c"} {
		row := models.Block{
			Name:   space,
			Type:   models.BlockTypeHTML,
			Space:  space,
			Status: models.StatusActive,
			Sort:   i,
		}
		if err := db.Omit("Spaces").Create(&row).Error; err != nil {
			t.Fatal(err)
		}
		if err := db.Create(&models.BlockSpace{
			BlockID: row.BlockID,
			Space:   space,
			Sort:    i,
		}).Error; err != nil {
			t.Fatal(err)
		}
	}
	for _, space := range []string{"a", "b", "c"} {
		if _, err := cache.BlocksForSpace(ctx, db, space); err != nil {
			t.Fatal(err)
		}
	}
	// Evict "a", delete it from DB, then "b"/"c" should still be served from cache
	// while "a" reloads empty.
	if err := db.Where("space = ?", "a").Delete(&models.Block{}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Where("space = ?", "b").Delete(&models.Block{}).Error; err != nil {
		t.Fatal(err)
	}
	bRows, err := cache.BlocksForSpace(ctx, db, "b")
	if err != nil {
		t.Fatal(err)
	}
	if len(bRows) != 1 {
		t.Fatalf("expected cached space b, got %d", len(bRows))
	}
	aRows, err := cache.BlocksForSpace(ctx, db, "a")
	if err != nil {
		t.Fatal(err)
	}
	if len(aRows) != 0 {
		t.Fatalf("expected evicted space a to reload empty, got %d", len(aRows))
	}
}
