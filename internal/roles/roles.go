package roles

import (
	"context"

	"github.com/rob121/cannon/internal/groups"
	"github.com/rob121/cannon/internal/models"
	"github.com/rob121/cannon/internal/security"
	"github.com/rob121/cannon/internal/sites"
	"gorm.io/gorm"
)

const (
	AdminRole   = security.RoleLegacyAdmin
	ManagerRole = security.RoleManager
	EditorRole  = security.RoleEditor
	WriterRole  = security.RoleWriter
	AuthorRole  = security.RoleLegacyAuthor
)

// EnsureDefaults seeds permissions, roles, and default groups.
func EnsureDefaults(db *gorm.DB) error {
	if err := security.EnsureForSite(db); err != nil {
		return err
	}
	if err := ensureDefaultGroups(db); err != nil {
		return err
	}
	return groups.EnsureDefaults(db)
}

func ensureDefaultGroups(db *gorm.DB) error {
	roleNames := map[string]string{
		groups.AdministratorsGroupName: security.RoleAdministrator,
		groups.ManagerGroupName:        security.RoleManager,
		groups.EditorGroupName:         security.RoleEditor,
		groups.WriterGroupName:         security.RoleWriter,
	}
	rolesByName := map[string]models.Role{}
	for _, name := range []string{
		security.RoleAdministrator,
		security.RoleManager,
		security.RoleEditor,
		security.RoleWriter,
	} {
		var role models.Role
		if err := db.Where("name = ?", name).First(&role).Error; err != nil {
			return err
		}
		rolesByName[name] = role
	}
	for groupName, roleName := range roleNames {
		role := rolesByName[roleName]
		if _, err := ensureGroup(db, groupName, models.GroupKindBackend, nil, []models.Role{role}); err != nil {
			return err
		}
	}
	return nil
}

func ensureGroup(db *gorm.DB, name string, kind models.GroupKind, parentID *uint, roles []models.Role) (models.Group, error) {
	var group models.Group
	err := db.Where("name = ?", name).First(&group).Error
	if err == gorm.ErrRecordNotFound {
		group = models.Group{Name: name, Kind: kind, ParentID: parentID, Status: models.StatusActive}
		if err := db.Create(&group).Error; err != nil {
			return models.Group{}, err
		}
	} else if err != nil {
		return models.Group{}, err
	} else {
		updates := map[string]any{"kind": kind, "parent_id": parentID}
		if err := db.Model(&group).Updates(updates).Error; err != nil {
			return models.Group{}, err
		}
	}
	if len(roles) > 0 {
		if err := db.Model(&group).Association("Roles").Replace(roles); err != nil {
			return models.Group{}, err
		}
	}
	return group, nil
}

// AssignAdmin adds a user to the Administrators group.
func AssignAdmin(ctx context.Context, userID uint) error {
	db, err := sites.DB(ctx)
	if err != nil {
		return err
	}
	var group models.Group
	if err := db.Where("name = ?", groups.AdministratorsGroupName).First(&group).Error; err != nil {
		return err
	}
	var u models.User
	if err := db.First(&u, userID).Error; err != nil {
		return err
	}
	if err := db.Model(&u).Association("Groups").Append(&group); err != nil {
		return err
	}
	security.InvalidateSiteUser(ctx, userID)
	return nil
}
