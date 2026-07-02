package router_test

import (
	"testing"

	"github.com/rob121/cannon/internal/models"
	"github.com/rob121/cannon/internal/router"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func openRouteTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&models.Route{}); err != nil {
		t.Fatal(err)
	}
	return db
}

func TestSetRouteDefaultExclusive(t *testing.T) {
	db := openRouteTestDB(t)
	first := models.Route{Name: "Home", Path: "/", Type: models.RouteTypeController, Status: models.StatusActive, IsDefault: true}
	second := models.Route{Name: "Blog", Path: "/blog", Type: models.RouteTypeController, Status: models.StatusActive}
	if err := db.Create(&first).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&second).Error; err != nil {
		t.Fatal(err)
	}

	if err := router.SetRouteDefault(db, second.RouteID); err != nil {
		t.Fatal(err)
	}

	var routes []models.Route
	if err := db.Order("route_id asc").Find(&routes).Error; err != nil {
		t.Fatal(err)
	}
	if routes[0].IsDefault {
		t.Fatalf("first route should not be default: %+v", routes[0])
	}
	if !routes[1].IsDefault {
		t.Fatalf("second route should be default: %+v", routes[1])
	}
}

func TestEnsureRouteDefaultLeavesUnset(t *testing.T) {
	db := openRouteTestDB(t)
	home := models.Route{Name: "Home", Path: "/", Type: models.RouteTypeController, Status: models.StatusActive, Controller: "content", ControllerAction: "index"}
	other := models.Route{Name: "Blog", Path: "/blog", Type: models.RouteTypeController, Status: models.StatusActive}
	if err := db.Create(&home).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&other).Error; err != nil {
		t.Fatal(err)
	}

	if err := router.EnsureRouteDefault(db); err != nil {
		t.Fatal(err)
	}

	var count int64
	if err := db.Model(&models.Route{}).Where("is_default = ?", true).Count(&count).Error; err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatalf("expected no auto-assigned default, got %d", count)
	}
}
