package security

import (
	"encoding/json"

	"github.com/rob121/cannon/internal/models"
	"gorm.io/gorm"
)

const systemRoleDirectPermissionsRevision = 1

type securityMigrationState struct {
	SystemRoleDirectPermissionsRev int `json:"system_role_direct_permissions_rev"`
}

// reconcileSystemRoleDirectPermissions resets direct grants on built-in roles to match code defaults.
func reconcileSystemRoleDirectPermissions(db *gorm.DB) error {
	roleByName := map[string]models.Role{}
	for _, def := range RegisteredRoleDefs() {
		if !def.SystemRole {
			continue
		}
		var role models.Role
		if err := db.Where("name = ?", def.Name).First(&role).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				continue
			}
			return err
		}
		roleByName[def.Name] = role
	}
	for _, def := range RegisteredRoleDefs() {
		if !def.SystemRole {
			continue
		}
		role, ok := roleByName[def.Name]
		if !ok {
			continue
		}
		if err := SaveRolePermissions(db, role.RoleID, def.Permissions, nil); err != nil {
			return err
		}
		var parentIDs []uint
		for _, parentName := range def.Inherits {
			if parent, ok := roleByName[parentName]; ok {
				parentIDs = append(parentIDs, parent.RoleID)
			}
		}
		if err := SaveRoleInheritance(db, role.RoleID, parentIDs); err != nil {
			return err
		}
	}
	return nil
}

func migrateSystemRoleDirectPermissions(db *gorm.DB) error {
	rev, err := loadSecurityMigrationRevision(db)
	if err != nil {
		return err
	}
	if rev >= systemRoleDirectPermissionsRevision {
		return nil
	}
	if err := reconcileSystemRoleDirectPermissions(db); err != nil {
		return err
	}
	return saveSecurityMigrationRevision(db, systemRoleDirectPermissionsRevision)
}

func loadSecurityMigrationRevision(db *gorm.DB) (int, error) {
	var row models.Setting
	err := db.Where("scope = ? AND section = ?", "global", "security_migrations").First(&row).Error
	if err == gorm.ErrRecordNotFound {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	var state securityMigrationState
	if row.Data == "" {
		return 0, nil
	}
	if err := json.Unmarshal([]byte(row.Data), &state); err != nil {
		return 0, err
	}
	return state.SystemRoleDirectPermissionsRev, nil
}

func saveSecurityMigrationRevision(db *gorm.DB, rev int) error {
	state := securityMigrationState{SystemRoleDirectPermissionsRev: rev}
	raw, err := json.Marshal(state)
	if err != nil {
		return err
	}
	var row models.Setting
	err = db.Where("scope = ? AND section = ?", "global", "security_migrations").First(&row).Error
	if err == gorm.ErrRecordNotFound {
		row = models.Setting{Scope: "global", Section: "security_migrations", Data: string(raw)}
		return db.Create(&row).Error
	}
	if err != nil {
		return err
	}
	row.Data = string(raw)
	return db.Save(&row).Error
}

// SystemRoleDirectPermissionsRevisionForTest exposes the migration revision for tests.
func SystemRoleDirectPermissionsRevisionForTest() int {
	return systemRoleDirectPermissionsRevision
}

// ParseSecurityMigrationRevisionForTest parses stored migration JSON.
func ParseSecurityMigrationRevisionForTest(data string) (int, error) {
	if data == "" {
		return 0, nil
	}
	var state securityMigrationState
	if err := json.Unmarshal([]byte(data), &state); err != nil {
		return 0, err
	}
	return state.SystemRoleDirectPermissionsRev, nil
}

// FormatSecurityMigrationRevisionForTest builds migration JSON.
func FormatSecurityMigrationRevisionForTest(rev int) string {
	raw, _ := json.Marshal(securityMigrationState{SystemRoleDirectPermissionsRev: rev})
	return string(raw)
}

// ReconcileSystemRoleDirectPermissionsForTest runs reconciliation without bumping revision.
func ReconcileSystemRoleDirectPermissionsForTest(db *gorm.DB) error {
	return reconcileSystemRoleDirectPermissions(db)
}
