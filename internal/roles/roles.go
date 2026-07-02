package roles

import (
	"context"

	"github.com/rob121/cannon/internal/groups"
	"github.com/rob121/cannon/internal/models"
	"github.com/rob121/cannon/internal/sites"
	"gorm.io/gorm"
)

const (
	AdminRole   = "admin"
	ManagerRole = "manager"
	EditorRole  = "editor"
	WriterRole  = "writer"
	AuthorRole  = "author"
)

// HasRole checks whether a user has a role via group membership.
func HasRole(ctx context.Context, userID uint, roleName string) (bool, error) {
	db, err := sites.DB(ctx)
	if err != nil {
		return false, err
	}

	var count int64
	err = db.Model(&models.User{}).
		Joins("JOIN user_groups ON user_groups.user_user_id = users.user_id").
		Joins("JOIN group_roles ON group_roles.group_group_id = user_groups.group_group_id").
		Joins("JOIN roles ON roles.role_id = group_roles.role_role_id").
		Where("users.user_id = ? AND roles.name = ? AND roles.status = ? AND users.status = ?",
			userID, roleName, models.StatusActive, models.StatusActive).
		Count(&count).Error
	return count > 0, err
}

// EnsureDefaults seeds roles and the editorial group hierarchy.
func EnsureDefaults(db *gorm.DB) error {
	roleNames := []string{AdminRole, ManagerRole, EditorRole, WriterRole, AuthorRole}
	rolesByName := map[string]models.Role{}
	for _, name := range roleNames {
		role, err := ensureRole(db, name)
		if err != nil {
			return err
		}
		rolesByName[name] = role
	}

	adminGroup, err := ensureGroup(db, groups.AdministratorsGroupName, models.GroupKindBackend, nil, []models.Role{rolesByName[AdminRole]})
	if err != nil {
		return err
	}
	managerGroup, err := ensureGroup(db, groups.ManagerGroupName, models.GroupKindBackend, &adminGroup.GroupID, []models.Role{rolesByName[ManagerRole]})
	if err != nil {
		return err
	}
	editorGroup, err := ensureGroup(db, groups.EditorGroupName, models.GroupKindBackend, &managerGroup.GroupID, []models.Role{rolesByName[EditorRole]})
	if err != nil {
		return err
	}
	if _, err := ensureGroup(db, groups.WriterGroupName, models.GroupKindBackend, &editorGroup.GroupID, []models.Role{rolesByName[WriterRole]}); err != nil {
		return err
	}

	return groups.EnsureDefaults(db)
}

func ensureRole(db *gorm.DB, name string) (models.Role, error) {
	var role models.Role
	if err := db.Where("name = ?", name).First(&role).Error; err == gorm.ErrRecordNotFound {
		role = models.Role{Name: name, Status: models.StatusActive}
		if err := db.Create(&role).Error; err != nil {
			return models.Role{}, err
		}
		return role, nil
	} else if err != nil {
		return models.Role{}, err
	}
	return role, nil
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
		updates := map[string]any{"kind": kind}
		if parentID != nil {
			updates["parent_id"] = *parentID
		}
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
	return db.Model(&u).Association("Groups").Append(&group)
}
