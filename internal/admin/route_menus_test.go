package admin

import (
	"testing"

	"github.com/rob121/cannon/internal/models"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func openAdminTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&models.Menu{}, &models.MenuItem{}, &models.Route{}); err != nil {
		t.Fatal(err)
	}
	return db
}

func TestAddRouteToMenuCreatesItem(t *testing.T) {
	db := openAdminTestDB(t)

	menu := models.Menu{MenuName: "main", Status: models.StatusActive}
	route := models.Route{Name: "Contact", Path: "/contact", Type: models.RouteTypeController, Status: models.StatusActive, Controller: "content", ControllerAction: "index"}
	if err := db.Create(&menu).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&route).Error; err != nil {
		t.Fatal(err)
	}

	added, err := addRouteToMenu(db, route, menu.MenuID, nil, "Contact Us")
	if err != nil {
		t.Fatal(err)
	}
	if !added {
		t.Fatal("expected menu item to be created")
	}

	var item models.MenuItem
	if err := db.Where("menu_id = ? AND route_id = ?", menu.MenuID, route.RouteID).First(&item).Error; err != nil {
		t.Fatal(err)
	}
	if item.Name != "Contact Us" {
		t.Fatalf("item name = %q", item.Name)
	}
	if item.RouteID == nil || *item.RouteID != route.RouteID {
		t.Fatalf("route_id = %v", item.RouteID)
	}

	addedAgain, err := addRouteToMenu(db, route, menu.MenuID, nil, "Duplicate")
	if err != nil {
		t.Fatal(err)
	}
	if addedAgain {
		t.Fatal("expected duplicate menu item to be skipped")
	}
}

func TestNextMenuItemSort(t *testing.T) {
	db := openAdminTestDB(t)

	menu := models.Menu{MenuName: "nav", Status: models.StatusActive}
	if err := db.Create(&menu).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&models.MenuItem{MenuID: menu.MenuID, Name: "One", Sort: 3}).Error; err != nil {
		t.Fatal(err)
	}
	if got := nextMenuItemSort(db, menu.MenuID, nil); got != 4 {
		t.Fatalf("next sort = %d, want 4", got)
	}
}
