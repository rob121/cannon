package database

import (
	"database/sql"
	"strings"

	"gorm.io/gorm"
)

// RepairMenuSchema fixes an inverted foreign key GORM previously created on menus.
// The child table menu_items should reference menus, not the other way around.
func RepairMenuSchema(db *gorm.DB) error {
	if db == nil || db.Dialector.Name() != "sqlite" {
		return nil
	}
	var ddl string
	if err := db.Raw("SELECT sql FROM sqlite_master WHERE type = 'table' AND name = 'menus'").Scan(&ddl).Error; err != nil {
		return err
	}
	if ddl == "" || !menusHasInvertedForeignKey(ddl) {
		return nil
	}

	return repairSQLiteTable(db, repairTableSpec{
		name:    "menus",
		columns: "menu_id, menu_name, status",
		copySQL: "menu_id, menu_name, status",
		indexes: []string{
			`CREATE UNIQUE INDEX IF NOT EXISTS idx_menus_menu_name ON menus(menu_name)`,
		},
	})
}

// RepairRoutesSchema fixes an inverted foreign key GORM previously created on routes.
// menu_items.route_id should reference routes, not the other way around.
func RepairRoutesSchema(db *gorm.DB) error {
	if db == nil || db.Dialector.Name() != "sqlite" {
		return nil
	}
	var ddl string
	if err := db.Raw("SELECT sql FROM sqlite_master WHERE type = 'table' AND name = 'routes'").Scan(&ddl).Error; err != nil {
		return err
	}
	if ddl == "" || !routesHasInvertedForeignKey(ddl) {
		return nil
	}

	return repairSQLiteTable(db, repairTableSpec{
		name:    "routes",
		columns: `route_id, name, path, type, status, target, extension_name, extension_page_id, extension_endpoint_id, metadata, controller, controller_action`,
		copySQL: `route_id, name, path, type, status, target, extension_name, extension_page_id, extension_endpoint_id, metadata, controller, controller_action`,
		indexes: []string{
			`CREATE UNIQUE INDEX IF NOT EXISTS idx_routes_path ON routes(path)`,
		},
	})
}

type repairTableSpec struct {
	name    string
	columns string
	copySQL string
	indexes []string
}

func repairSQLiteTable(db *gorm.DB, spec repairTableSpec) error {
	sqlDB, err := db.DB()
	if err != nil {
		return err
	}
	if err := setSQLiteForeignKeys(sqlDB, false); err != nil {
		return err
	}
	defer setSQLiteForeignKeys(sqlDB, true)

	repairName := spec.name + "__repair"
	return db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec(`
			CREATE TABLE ` + repairName + ` (
				` + repairColumnsDDL(spec.name) + `
			);
		`).Error; err != nil {
			return err
		}
		if err := tx.Exec(`
			INSERT INTO ` + repairName + ` (` + spec.columns + `)
			SELECT ` + spec.copySQL + ` FROM ` + spec.name + `;
		`).Error; err != nil {
			return err
		}
		if err := tx.Exec(`DROP TABLE ` + spec.name).Error; err != nil {
			return err
		}
		if err := tx.Exec(`ALTER TABLE ` + repairName + ` RENAME TO ` + spec.name).Error; err != nil {
			return err
		}
		for _, idx := range spec.indexes {
			if err := tx.Exec(idx).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func repairColumnsDDL(table string) string {
	switch table {
	case "menus":
		return `menu_id INTEGER PRIMARY KEY AUTOINCREMENT,
				menu_name TEXT NOT NULL,
				status TEXT NOT NULL DEFAULT 'active'`
	case "routes":
		return `route_id INTEGER PRIMARY KEY AUTOINCREMENT,
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
				controller_action TEXT`
	default:
		return ""
	}
}

func setSQLiteForeignKeys(sqlDB *sql.DB, enabled bool) error {
	value := "OFF"
	if enabled {
		value = "ON"
	}
	_, err := sqlDB.Exec("PRAGMA foreign_keys=" + value)
	return err
}

func menusHasInvertedForeignKey(sql string) bool {
	lower := strings.ToLower(sql)
	return strings.Contains(lower, "references `menu_items`") ||
		strings.Contains(lower, "references menu_items")
}

func routesHasInvertedForeignKey(sql string) bool {
	lower := strings.ToLower(sql)
	return strings.Contains(lower, "references `menu_items`") ||
		strings.Contains(lower, "references menu_items")
}
