package database

import (
	"gorm.io/gorm"
)

// RepairRoleInheritanceSchema renames legacy GORM-pluralized role_inheritances tables.
func RepairRoleInheritanceSchema(db *gorm.DB) error {
	if db == nil {
		return nil
	}
	migrator := db.Migrator()
	if !migrator.HasTable("role_inheritances") {
		return nil
	}
	if migrator.HasTable("role_inheritance") {
		return migrator.DropTable("role_inheritances")
	}
	return migrator.RenameTable("role_inheritances", "role_inheritance")
}
