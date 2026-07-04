package security

import (
	"github.com/rob121/cannon/internal/models"
	"gorm.io/gorm"
)

// SyncToDB upserts registered permissions and marks removed ones deprecated.
func SyncToDB(db *gorm.DB) error {
	registered := RegisteredPermissions()
	active := map[string]struct{}{}
	for _, p := range registered {
		active[p.ID] = struct{}{}
		if err := upsertPermission(db, p); err != nil {
			return err
		}
	}
	var existing []models.Permission
	if err := db.Find(&existing).Error; err != nil {
		return err
	}
	for _, row := range existing {
		if _, ok := active[row.Key]; ok {
			continue
		}
		if err := db.Model(&row).Update("deprecated", true).Error; err != nil {
			return err
		}
	}
	return nil
}

func upsertPermission(db *gorm.DB, p Permission) error {
	var row models.Permission
	err := db.Where("key = ?", p.ID).First(&row).Error
	if err == gorm.ErrRecordNotFound {
		row = models.Permission{
			Key:         p.ID,
			DisplayName: p.DisplayName,
			Description: p.Description,
			Category:    p.Category,
			Dangerous:   p.Dangerous,
		}
		return db.Create(&row).Error
	}
	if err != nil {
		return err
	}
	return db.Model(&row).Updates(map[string]any{
		"display_name": p.DisplayName,
		"description":  p.Description,
		"category":     p.Category,
		"dangerous":    p.Dangerous,
		"deprecated":   false,
	}).Error
}

// SeedDefaults creates system roles, inheritance, and default permission assignments.
func SeedDefaults(db *gorm.DB) error {
	roleByName := map[string]models.Role{}
	for _, def := range RegisteredRoleDefs() {
		role, err := ensureDBRole(db, def)
		if err != nil {
			return err
		}
		roleByName[def.Name] = role
	}
	for _, def := range RegisteredRoleDefs() {
		role := roleByName[def.Name]
		var parentIDs []uint
		for _, parentName := range def.Inherits {
			if parent, ok := roleByName[parentName]; ok {
				parentIDs = append(parentIDs, parent.RoleID)
			}
		}
		if def.SystemRole {
			if err := SaveRoleInheritance(db, role.RoleID, parentIDs); err != nil {
				return err
			}
		}
		var count int64
		if err := db.Model(&models.RolePermission{}).Where("role_id = ?", role.RoleID).Count(&count).Error; err != nil {
			return err
		}
		if count == 0 && len(def.Permissions) > 0 {
			if err := SaveRolePermissions(db, role.RoleID, def.Permissions, nil); err != nil {
				return err
			}
		}
	}
	return migrateLegacy(db, roleByName)
}

func ensureDBRole(db *gorm.DB, def RoleDef) (models.Role, error) {
	var role models.Role
	err := db.Where("name = ?", def.Name).First(&role).Error
	if err == gorm.ErrRecordNotFound {
		role = models.Role{
			Name:        def.Name,
			Description: def.Description,
			SystemRole:  def.SystemRole,
			Status:      models.StatusActive,
		}
		if err := db.Create(&role).Error; err != nil {
			return models.Role{}, err
		}
		return role, nil
	}
	if err != nil {
		return models.Role{}, err
	}
	updates := map[string]any{
		"description": def.Description,
		"system_role": def.SystemRole,
	}
	if err := db.Model(&role).Updates(updates).Error; err != nil {
		return models.Role{}, err
	}
	return role, nil
}

// EnsureForSite syncs permissions and seeds defaults for one site database.
func EnsureForSite(db *gorm.DB) error {
	if err := SyncToDB(db); err != nil {
		return err
	}
	return SeedDefaults(db)
}
