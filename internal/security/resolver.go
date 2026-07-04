package security

import (
	"context"
	"fmt"

	"github.com/rob121/cannon/internal/models"
	"github.com/rob121/cannon/internal/sites"
	"gorm.io/gorm"
)

// ResolveEffective loads and expands permissions for a user.
func ResolveEffective(ctx context.Context, userID uint) (EffectivePermissions, error) {
	if userID == 0 {
		return NewEffectivePermissions(), nil
	}
	site, err := sites.FromContext(ctx)
	if err != nil {
		return EffectivePermissions{}, err
	}
	if perms, ok := cacheGet(site.ID, userID); ok {
		return perms, nil
	}
	db, err := sites.DB(ctx)
	if err != nil {
		return EffectivePermissions{}, err
	}
	perms, err := resolveFromDB(db, userID)
	if err != nil {
		return EffectivePermissions{}, err
	}
	cacheSet(site.ID, userID, perms)
	return perms, nil
}

func resolveFromDB(db *gorm.DB, userID uint) (EffectivePermissions, error) {
	roleIDs, err := collectRoleIDs(db, userID)
	if err != nil {
		return EffectivePermissions{}, err
	}
	if len(roleIDs) == 0 {
		return NewEffectivePermissions(), nil
	}
	expanded, err := expandRoleInheritance(db, roleIDs)
	if err != nil {
		return EffectivePermissions{}, err
	}
	return loadRolePermissions(db, expanded)
}

func collectRoleIDs(db *gorm.DB, userID uint) ([]uint, error) {
	seen := map[uint]struct{}{}
	var ids []uint

	var direct []uint
	if err := db.Model(&models.User{}).
		Joins("JOIN user_roles ON user_roles.user_user_id = users.user_id").
		Joins("JOIN roles ON roles.role_id = user_roles.role_role_id").
		Where("users.user_id = ? AND users.status = ? AND roles.status = ?",
			userID, models.StatusActive, models.StatusActive).
		Pluck("roles.role_id", &direct).Error; err != nil {
		return nil, err
	}
	for _, id := range direct {
		if _, ok := seen[id]; !ok {
			seen[id] = struct{}{}
			ids = append(ids, id)
		}
	}

	var viaGroup []uint
	if err := db.Model(&models.User{}).
		Joins("JOIN user_groups ON user_groups.user_user_id = users.user_id").
		Joins("JOIN groups ON groups.group_id = user_groups.group_group_id").
		Joins("JOIN group_roles ON group_roles.group_group_id = groups.group_id").
		Joins("JOIN roles ON roles.role_id = group_roles.role_role_id").
		Where("users.user_id = ? AND users.status = ? AND groups.status = ? AND roles.status = ?",
			userID, models.StatusActive, models.StatusActive, models.StatusActive).
		Pluck("roles.role_id", &viaGroup).Error; err != nil {
		return nil, err
	}
	for _, id := range viaGroup {
		if _, ok := seen[id]; !ok {
			seen[id] = struct{}{}
			ids = append(ids, id)
		}
	}
	return ids, nil
}

func expandRoleInheritance(db *gorm.DB, roleIDs []uint) ([]uint, error) {
	seen := map[uint]struct{}{}
	queue := append([]uint(nil), roleIDs...)
	for len(queue) > 0 {
		id := queue[0]
		queue = queue[1:]
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		var parents []uint
		if err := db.Model(&models.RoleInheritance{}).
			Where("child_role_id = ?", id).
			Pluck("parent_role_id", &parents).Error; err != nil {
			return nil, err
		}
		queue = append(queue, parents...)
	}
	out := make([]uint, 0, len(seen))
	for id := range seen {
		out = append(out, id)
	}
	return out, nil
}

func loadRolePermissions(db *gorm.DB, roleIDs []uint) (EffectivePermissions, error) {
	out := NewEffectivePermissions()
	if len(roleIDs) == 0 {
		return out, nil
	}
	var rows []models.RolePermission
	if err := db.Where("role_id IN ?", roleIDs).Find(&rows).Error; err != nil {
		return EffectivePermissions{}, err
	}
	for _, row := range rows {
		if row.Denied {
			out.Deny[row.PermissionKey] = struct{}{}
		} else {
			out.Allow[row.PermissionKey] = struct{}{}
		}
	}
	return out, nil
}

// ValidateInheritance ensures adding parent→child links would not create a cycle.
func ValidateInheritance(db *gorm.DB, childRoleID uint, parentRoleIDs []uint) error {
	for _, parentID := range parentRoleIDs {
		if parentID == childRoleID {
			return fmt.Errorf("role cannot inherit from itself")
		}
		if wouldCycle(db, childRoleID, parentID) {
			return fmt.Errorf("role inheritance cycle detected")
		}
	}
	return nil
}

func wouldCycle(db *gorm.DB, childRoleID, parentRoleID uint) bool {
	visited := map[uint]struct{}{}
	queue := []uint{parentRoleID}
	for len(queue) > 0 {
		id := queue[0]
		queue = queue[1:]
		if id == childRoleID {
			return true
		}
		if _, ok := visited[id]; ok {
			continue
		}
		visited[id] = struct{}{}
		var parents []uint
		if err := db.Model(&models.RoleInheritance{}).
			Where("child_role_id = ?", id).
			Pluck("parent_role_id", &parents).Error; err != nil {
			return true
		}
		queue = append(queue, parents...)
	}
	return false
}

// SaveRoleInheritance replaces inheritance links for a role.
func SaveRoleInheritance(db *gorm.DB, childRoleID uint, parentRoleIDs []uint) error {
	if err := ValidateInheritance(db, childRoleID, parentRoleIDs); err != nil {
		return err
	}
	if err := db.Where("child_role_id = ?", childRoleID).Delete(&models.RoleInheritance{}).Error; err != nil {
		return err
	}
	for _, parentID := range parentRoleIDs {
		if parentID == 0 {
			continue
		}
		row := models.RoleInheritance{ChildRoleID: childRoleID, ParentRoleID: parentID}
		if err := db.Create(&row).Error; err != nil {
			return err
		}
	}
	return nil
}

// LoadParentRoleIDs returns parent role ids for a child role.
func LoadParentRoleIDs(db *gorm.DB, childRoleID uint) ([]uint, error) {
	var ids []uint
	err := db.Model(&models.RoleInheritance{}).
		Where("child_role_id = ?", childRoleID).
		Pluck("parent_role_id", &ids).Error
	return ids, err
}

// SaveRolePermissions replaces direct permissions assigned to a role.
func SaveRolePermissions(db *gorm.DB, roleID uint, allows, denies []string) error {
	if err := db.Where("role_id = ?", roleID).Delete(&models.RolePermission{}).Error; err != nil {
		return err
	}
	seen := map[string]struct{}{}
	for _, key := range allows {
		if key == "" {
			continue
		}
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		row := models.RolePermission{RoleID: roleID, PermissionKey: key, Denied: false}
		if err := db.Create(&row).Error; err != nil {
			return err
		}
	}
	for _, key := range denies {
		if key == "" {
			continue
		}
		if _, ok := seen[key]; ok {
			return fmt.Errorf("permission %q cannot be both allowed and denied", key)
		}
		seen[key] = struct{}{}
		row := models.RolePermission{RoleID: roleID, PermissionKey: key, Denied: true}
		if err := db.Create(&row).Error; err != nil {
			return err
		}
	}
	return nil
}

// LoadRolePermissionKeys returns allowed permission keys directly on a role (not inherited).
func LoadRolePermissionKeys(db *gorm.DB, roleID uint) ([]string, error) {
	allows, _, err := LoadRolePermissionAssignments(db, roleID)
	return allows, err
}

// LoadRolePermissionAssignments returns allow and deny keys directly on a role.
func LoadRolePermissionAssignments(db *gorm.DB, roleID uint) ([]string, []string, error) {
	var rows []models.RolePermission
	if err := db.Where("role_id = ?", roleID).Order("permission_key asc").Find(&rows).Error; err != nil {
		return nil, nil, err
	}
	allows := make([]string, 0, len(rows))
	denies := make([]string, 0, len(rows))
	for _, row := range rows {
		if row.Denied {
			denies = append(denies, row.PermissionKey)
		} else {
			allows = append(allows, row.PermissionKey)
		}
	}
	return allows, denies, nil
}
