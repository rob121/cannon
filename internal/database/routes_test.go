package database_test

import (
	"strings"
	"testing"

	"github.com/rob121/cannon/internal/config"
	"github.com/rob121/cannon/internal/database"
	"github.com/rob121/cannon/internal/models"
	"gorm.io/gorm"
)

func TestRouteRemigrateAfterMenuItems(t *testing.T) {
	path := t.TempDir() + "/reload.sqlite"
	site := config.SiteConfig{
		ID: "test",
		Database: config.DatabaseConfig{
			Type: "sqlite",
			DSN:  path,
		},
	}
	if err := database.Migrate(&site); err != nil {
		t.Fatalf("initial migrate: %v", err)
	}
	db, err := database.Get("test")
	if err != nil {
		t.Fatal(err)
	}
	assertRoutesSchema(t, db)

	if err := db.AutoMigrate(&models.Route{}); err != nil {
		t.Fatalf("remigrate routes: %v", err)
	}
	assertRoutesSchema(t, db)
}

func TestRouteMigrateWithInvertedForeignKey(t *testing.T) {
	db := openTestSQLite(t)
	if err := db.Exec(`
		CREATE TABLE routes (
			route_id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			path TEXT NOT NULL,
			type TEXT NOT NULL,
			status TEXT NOT NULL DEFAULT 'active',
			target TEXT,
			extension_name TEXT,
			extension_page_id TEXT,
			extension_endpoint_id TEXT,
			metadata TEXT,
			controller TEXT,
			controller_action TEXT,
			CONSTRAINT fk_menu_items_route FOREIGN KEY (route_id) REFERENCES menu_items(menu_item_id)
		);
		CREATE UNIQUE INDEX idx_routes_path ON routes(path);
		CREATE TABLE menu_items (
			menu_item_id INTEGER PRIMARY KEY AUTOINCREMENT,
			menu_id INTEGER NOT NULL,
			name TEXT NOT NULL,
			route_id INTEGER,
			class TEXT,
			target TEXT,
			sort INTEGER NOT NULL DEFAULT 0
		);
	`).Error; err != nil {
		t.Fatal(err)
	}

	if err := database.RepairRoutesSchema(db); err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&models.Route{}); err != nil {
		t.Fatalf("automigrate route: %v", err)
	}
	assertRoutesSchema(t, db)
}

func TestRoutesSchemaHasNoMenuItemsFK(t *testing.T) {
	db := openTestSQLite(t)
	migrateAll(t, db)
	assertRoutesSchema(t, db)
}

func assertRoutesSchema(t *testing.T, db *gorm.DB) {
	t.Helper()
	var sql string
	if err := db.Raw("SELECT sql FROM sqlite_master WHERE type = 'table' AND name = 'routes'").Scan(&sql).Error; err != nil {
		t.Fatal(err)
	}
	if sql == "" {
		t.Fatal("routes table missing")
	}
	lower := strings.ToLower(sql)
	if strings.Contains(lower, "references menu_items") || strings.Contains(lower, "references `menu_items`") {
		t.Fatalf("routes table has inverted foreign key: %s", sql)
	}
}
