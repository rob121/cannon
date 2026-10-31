package security

import (
	"context"
	"fmt"
	"strings"

	"github.com/rob121/cannon/internal/models"
	"gorm.io/gorm"
)

const extensionInstallerRolePrefix = "extension-installer:"

// GrantExtensionToInstaller ensures the user can open the extension admin UI after
// installing it. Extensions with registered permissions require explicit allows;
// this creates or updates a dedicated role and assigns it only to the installer.
func GrantExtensionToInstaller(ctx context.Context, db *gorm.DB, userID uint, extensionName string) error {
	extensionName = strings.TrimSpace(extensionName)
	if userID == 0 || extensionName == "" || db == nil {
		return nil
	}
	keys := ExtensionAdminPermissionIDs(extensionName)
	if len(keys) == 0 {
		return nil
	}
	effective, err := ResolveEffective(ctx, userID)
	if err != nil {
		return err
	}
	if HasExtensionAdminAccess(effective, extensionName) {
		return nil
	}
	role, err := ensureExtensionInstallerRole(db, extensionName)
	if err != nil {
		return err
	}
	if err := AppendRoleAllowPermissions(db, role.RoleID, keys); err != nil {
		return err
	}
	if err := assignUserRole(db, userID, role.RoleID); err != nil {
		return err
	}
	InvalidateSiteUser(ctx, userID)
	return nil
}

func extensionInstallerRoleName(extensionName string) string {
	name := extensionInstallerRolePrefix + extensionName
	if len(name) <= 128 {
		return name
	}
	return name[:128]
}

func ensureExtensionInstallerRole(db *gorm.DB, extensionName string) (models.Role, error) {
	name := extensionInstallerRoleName(extensionName)
	var role models.Role
	err := db.Where("name = ?", name).First(&role).Error
	if err == nil {
		return role, nil
	}
	if err != gorm.ErrRecordNotFound {
		return models.Role{}, err
	}
	role = models.Role{
		Name:        name,
		Description: fmt.Sprintf("Auto-assigned when a user installs %s. Attach this role to grant the same extension access to others.", extensionName),
		Status:      models.StatusActive,
		SystemRole:  false,
	}
	if err := db.Create(&role).Error; err != nil {
		return models.Role{}, err
	}
	return role, nil
}

// AppendRoleAllowPermissions adds allow keys to a role without removing existing assignments.
func AppendRoleAllowPermissions(db *gorm.DB, roleID uint, keys []string) error {
	if roleID == 0 || len(keys) == 0 {
		return nil
	}
	allows, denies, err := LoadRolePermissionAssignments(db, roleID)
	if err != nil {
		return err
	}
	allowSet := map[string]struct{}{}
	for _, key := range allows {
		allowSet[key] = struct{}{}
	}
	denySet := map[string]struct{}{}
	for _, key := range denies {
		denySet[key] = struct{}{}
	}
	for _, key := range keys {
		key = strings.TrimSpace(key)
		if key == "" {
			continue
		}
		delete(denySet, key)
		allowSet[key] = struct{}{}
	}
	newAllows := make([]string, 0, len(allowSet))
	for key := range allowSet {
		newAllows = append(newAllows, key)
	}
	newDenies := make([]string, 0, len(denySet))
	for key := range denySet {
		newDenies = append(newDenies, key)
	}
	return SaveRolePermissions(db, roleID, newAllows, newDenies)
}

func assignUserRole(db *gorm.DB, userID, roleID uint) error {
	var count int64
	if err := db.Table("user_roles").
		Where("user_user_id = ? AND role_role_id = ?", userID, roleID).
		Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return nil
	}
	return db.Exec(
		"INSERT INTO user_roles (user_user_id, role_role_id) VALUES (?, ?)",
		userID, roleID,
	).Error
}
