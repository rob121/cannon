package security

import (
	"strings"

	"github.com/rob121/cannon/internal/models"
	"gorm.io/gorm"
)

func migrateLegacy(db *gorm.DB, roleByName map[string]models.Role) error {
	if err := migrateLegacyRoleNames(db, roleByName); err != nil {
		return err
	}
	if err := migrateGroupAdminRoutes(db, roleByName); err != nil {
		return err
	}
	if err := flattenEditorialGroups(db); err != nil {
		return err
	}
	return migrateContentPermissionKeys(db)
}

func migrateLegacyRoleNames(db *gorm.DB, roleByName map[string]models.Role) error {
	legacyMap := map[string]string{
		RoleLegacyAdmin:  RoleAdministrator,
		RoleLegacyAuthor: RoleWriter,
		"manager":        RoleManager,
		"editor":         RoleEditor,
		"writer":         RoleWriter,
	}
	for oldName, newName := range legacyMap {
		newRole, ok := roleByName[newName]
		if !ok {
			continue
		}
		var oldRole models.Role
		if err := db.Where("name = ?", oldName).First(&oldRole).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				continue
			}
			return err
		}
		if oldRole.RoleID == newRole.RoleID {
			continue
		}
		if err := reassignRoleReferences(db, oldRole.RoleID, newRole.RoleID); err != nil {
			return err
		}
		if oldRole.SystemRole || oldName == RoleLegacyAdmin {
			if err := db.Delete(&oldRole).Error; err != nil {
				return err
			}
		}
	}
	return nil
}

func reassignRoleReferences(db *gorm.DB, oldID, newID uint) error {
	var groupIDs []uint
	if err := db.Table("group_roles").Where("role_role_id = ?", oldID).Pluck("group_group_id", &groupIDs).Error; err != nil {
		return err
	}
	for _, gid := range groupIDs {
		var count int64
		db.Table("group_roles").Where("group_group_id = ? AND role_role_id = ?", gid, newID).Count(&count)
		if count > 0 {
			_ = db.Exec("DELETE FROM group_roles WHERE group_group_id = ? AND role_role_id = ?", gid, oldID).Error
			continue
		}
		if err := db.Exec("UPDATE group_roles SET role_role_id = ? WHERE group_group_id = ? AND role_role_id = ?", newID, gid, oldID).Error; err != nil {
			return err
		}
	}
	if err := db.Exec("UPDATE user_roles SET role_role_id = ? WHERE role_role_id = ?", newID, oldID).Error; err != nil {
		return err
	}
	var perms []models.RolePermission
	if err := db.Where("role_id = ?", oldID).Find(&perms).Error; err != nil {
		return err
	}
	for _, p := range perms {
		row := models.RolePermission{RoleID: newID, PermissionKey: p.PermissionKey}
		_ = db.Where("role_id = ? AND permission_key = ?", newID, p.PermissionKey).
			FirstOrCreate(&row).Error
	}
	if err := db.Where("role_id = ?", oldID).Delete(&models.RolePermission{}).Error; err != nil {
		return err
	}
	if err := db.Exec(
		"UPDATE role_inheritance SET parent_role_id = ? WHERE parent_role_id = ?",
		newID, oldID,
	).Error; err != nil {
		return err
	}
	if err := db.Exec(
		"UPDATE role_inheritance SET child_role_id = ? WHERE child_role_id = ?",
		newID, oldID,
	).Error; err != nil {
		return err
	}
	return nil
}

func migrateGroupAdminRoutes(db *gorm.DB, roleByName map[string]models.Role) error {
	if !db.Migrator().HasTable("group_admin_routes") {
		return nil
	}
	type legacyRoute struct {
		GroupID  uint
		Path     string
		CanRead  bool
		CanWrite bool
	}
	var rows []legacyRoute
	if err := db.Table("group_admin_routes").Find(&rows).Error; err != nil {
		return nil
	}
	if len(rows) == 0 {
		return dropLegacyAdminRoutesTable(db)
	}
	for _, row := range rows {
		var group models.Group
		if err := db.Preload("Roles").First(&group, row.GroupID).Error; err != nil {
			continue
		}
		if len(group.Roles) == 0 {
			continue
		}
		section := strings.TrimPrefix(row.Path, "/")
		var keys []string
		if row.CanRead {
			keys = append(keys, "core.admin."+section+".read")
		}
		if row.CanWrite {
			keys = append(keys, "core.admin."+section+".write")
		}
		for _, role := range group.Roles {
			existing, _ := LoadRolePermissionKeys(db, role.RoleID)
			merged := mergeKeys(existing, keys)
			if err := SaveRolePermissions(db, role.RoleID, merged, nil); err != nil {
				return err
			}
		}
	}
	return dropLegacyAdminRoutesTable(db)
}

func dropLegacyAdminRoutesTable(db *gorm.DB) error {
	if db.Migrator().HasTable("group_admin_routes") {
		return db.Migrator().DropTable("group_admin_routes")
	}
	return nil
}

func mergeKeys(existing, add []string) []string {
	seen := map[string]struct{}{}
	var out []string
	for _, k := range append(existing, add...) {
		if k == "" {
			continue
		}
		if _, ok := seen[k]; ok {
			continue
		}
		seen[k] = struct{}{}
		out = append(out, k)
	}
	return out
}

func flattenEditorialGroups(db *gorm.DB) error {
	names := []string{"Administrators", "Manager", "Editor", "Writer"}
	for _, name := range names {
		if err := db.Model(&models.Group{}).
			Where("name = ?", name).
			Update("parent_id", nil).Error; err != nil {
			return err
		}
	}
	roleMap := map[string]string{
		"Administrators": RoleAdministrator,
		"Manager":        RoleManager,
		"Editor":         RoleEditor,
		"Writer":         RoleWriter,
	}
	for groupName, roleName := range roleMap {
		var group models.Group
		if err := db.Where("name = ?", groupName).First(&group).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				continue
			}
			return err
		}
		var role models.Role
		if err := db.Where("name = ?", roleName).First(&role).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				continue
			}
			return err
		}
		if err := db.Model(&group).Association("Roles").Replace([]models.Role{role}); err != nil {
			return err
		}
	}
	return nil
}

func migrateContentPermissionKeys(db *gorm.DB) error {
	var rows []models.RolePermission
	if err := db.Find(&rows).Error; err != nil {
		return err
	}
	for _, row := range rows {
		newKey, ok := legacyContentPermissionMap[row.PermissionKey]
		if !ok || newKey == row.PermissionKey {
			continue
		}
		var count int64
		if err := db.Model(&models.RolePermission{}).
			Where("role_id = ? AND permission_key = ?", row.RoleID, newKey).
			Count(&count).Error; err != nil {
			return err
		}
		if count > 0 {
			if err := db.Delete(&row).Error; err != nil {
				return err
			}
			continue
		}
		if err := db.Model(&row).Update("permission_key", newKey).Error; err != nil {
			return err
		}
	}
	return nil
}
