package database_test

import (
	"testing"

	"github.com/rob121/cannon/internal/database"
	"github.com/rob121/cannon/internal/models"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestRepairRoleInheritanceSchemaRenamesLegacyTable(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Exec(`CREATE TABLE role_inheritances (
		child_role_id INTEGER NOT NULL,
		parent_role_id INTEGER NOT NULL,
		PRIMARY KEY (child_role_id, parent_role_id)
	)`).Error; err != nil {
		t.Fatal(err)
	}
	if err := database.RepairRoleInheritanceSchema(db); err != nil {
		t.Fatal(err)
	}
	if !db.Migrator().HasTable("role_inheritance") {
		t.Fatal("expected role_inheritance table")
	}
	if db.Migrator().HasTable("role_inheritances") {
		t.Fatal("legacy role_inheritances table should be renamed")
	}
	if err := db.AutoMigrate(&models.RoleInheritance{}); err != nil {
		t.Fatal(err)
	}
}
