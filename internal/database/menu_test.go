package database_test

import (
	"strings"
	"testing"

	"github.com/rob121/cannon/internal/config"
	"github.com/rob121/cannon/internal/database"
	"github.com/rob121/cannon/internal/models"
	"gorm.io/gorm"
)

func TestMenuCreateAfterMigrate(t *testing.T) {
	db := openTestSQLite(t)
	if err := database.RepairMenuSchema(db); err != nil {
		t.Fatal(err)
	}
	migrateAll(t, db)
	assertMenusSchema(t, db)

	menu := models.Menu{MenuName: "main", Status: models.StatusActive}
	if err := db.Create(&menu).Error; err != nil {
		t.Fatalf("create menu: %v", err)
	}
	if menu.MenuID == 0 {
		t.Fatal("expected menu id")
	}
}

func TestRepairMenuSchemaFixesInvertedForeignKey(t *testing.T) {
	db := openTestSQLite(t)
	if err := db.Exec(`
		CREATE TABLE menus (
			menu_id INTEGER PRIMARY KEY AUTOINCREMENT,
			menu_name TEXT NOT NULL,
			status TEXT NOT NULL DEFAULT 'active',
			CONSTRAINT fk_menu_items_menu FOREIGN KEY (menu_id) REFERENCES menu_items(menu_item_id)
		);
		CREATE UNIQUE INDEX idx_menus_menu_name ON menus(menu_name);
		CREATE TABLE menu_items (
			menu_item_id INTEGER PRIMARY KEY AUTOINCREMENT,
			menu_id INTEGER NOT NULL,
			name TEXT NOT NULL,
			route_id INTEGER,
			class TEXT,
			target TEXT,
			sort INTEGER NOT NULL DEFAULT 0,
			CONSTRAINT fk_menus_items FOREIGN KEY (menu_id) REFERENCES menus(menu_id)
		);
	`).Error; err != nil {
		t.Fatal(err)
	}

	menu := models.Menu{MenuName: "main", Status: models.StatusActive}
	if err := db.Create(&menu).Error; err == nil {
		t.Fatal("expected create to fail with inverted foreign key")
	}

	if err := database.RepairMenuSchema(db); err != nil {
		t.Fatal(err)
	}
	assertMenusSchema(t, db)

	menu = models.Menu{MenuName: "main", Status: models.StatusActive}
	if err := db.Create(&menu).Error; err != nil {
		t.Fatalf("create menu after repair: %v", err)
	}

	var itemCount int64
	if err := db.Table("menu_items").Count(&itemCount).Error; err != nil {
		t.Fatal(err)
	}
	if itemCount != 0 {
		t.Fatalf("expected menu_items preserved, got %d rows", itemCount)
	}
}

func TestMigrateRepairsMenusBeforeAutoMigrate(t *testing.T) {
	path := t.TempDir() + "/broken.sqlite"
	db, err := database.Open(config.DatabaseConfig{Type: "sqlite", DSN: path})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Exec(`
		CREATE TABLE menus (
			menu_id INTEGER PRIMARY KEY AUTOINCREMENT,
			menu_name TEXT NOT NULL,
			status TEXT NOT NULL DEFAULT 'active',
			CONSTRAINT fk_menu_items_menu FOREIGN KEY (menu_id) REFERENCES menu_items(menu_item_id)
		);
		CREATE UNIQUE INDEX idx_menus_menu_name ON menus(menu_name);
		CREATE TABLE menu_items (
			menu_item_id INTEGER PRIMARY KEY AUTOINCREMENT,
			menu_id INTEGER NOT NULL,
			name TEXT NOT NULL,
			route_id INTEGER,
			class TEXT,
			target TEXT,
			sort INTEGER NOT NULL DEFAULT 0,
			CONSTRAINT fk_menus_items FOREIGN KEY (menu_id) REFERENCES menus(menu_id)
		);
	`).Error; err != nil {
		t.Fatal(err)
	}

	site := config.SiteConfig{
		ID: "test",
		Database: config.DatabaseConfig{
			Type: "sqlite",
			DSN:  path,
		},
	}
	database.CloseAll()
	if err := database.Migrate(&site); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	db, err = database.Get("test")
	if err != nil {
		t.Fatal(err)
	}
	assertMenusSchema(t, db)

	menu := models.Menu{MenuName: "main", Status: models.StatusActive}
	if err := db.Create(&menu).Error; err != nil {
		t.Fatalf("create menu: %v", err)
	}
}

func openTestSQLite(t *testing.T) *gorm.DB {
	t.Helper()
	path := t.TempDir() + "/test.sqlite"
	db, err := database.Open(config.DatabaseConfig{Type: "sqlite", DSN: path})
	if err != nil {
		t.Fatal(err)
	}
	return db
}

func migrateAll(t *testing.T, db *gorm.DB) {
	t.Helper()
	for _, model := range models.All() {
		if err := db.AutoMigrate(model); err != nil {
			t.Fatalf("migrate %T: %v", model, err)
		}
	}
}

func assertMenusSchema(t *testing.T, db *gorm.DB) {
	t.Helper()
	var sql string
	if err := db.Raw("SELECT sql FROM sqlite_master WHERE type = 'table' AND name = 'menus'").Scan(&sql).Error; err != nil {
		t.Fatal(err)
	}
	if sql == "" {
		t.Fatal("menus table missing")
	}
	lower := strings.ToLower(sql)
	if strings.Contains(lower, "references `menu_items`") || strings.Contains(lower, "references menu_items") {
		t.Fatalf("menus table has inverted foreign key: %s", sql)
	}
}
