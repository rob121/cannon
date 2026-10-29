package admin

import (
	"context"
	"fmt"

	"github.com/rob121/cannon/internal/models"
	"github.com/rob121/cannon/internal/security"
	"github.com/rob121/cannon/internal/user"
	"gorm.io/gorm"
)

func adminUsersWritePerm(write bool) string {
	if write {
		return security.AdminPermissionForPath("/users", true)
	}
	return security.AdminPermissionForPath("/users", false)
}

func canAdminUsersFullWrite(ctx context.Context, userID uint) (bool, error) {
	return security.Can(ctx, userID, adminUsersWritePerm(true))
}

// CanManageUserProfile reports whether the user may edit identity and profile fields.
func CanManageUserProfile(ctx context.Context, userID uint) (bool, error) {
	if ok, err := canAdminUsersFullWrite(ctx, userID); err != nil || ok {
		return ok, err
	}
	return security.Can(ctx, userID, security.PermAdminUsersProfileWrite)
}

// CanManageUserMembership reports whether the user may assign roles and groups.
func CanManageUserMembership(ctx context.Context, userID uint) (bool, error) {
	if ok, err := canAdminUsersFullWrite(ctx, userID); err != nil || ok {
		return ok, err
	}
	return security.Can(ctx, userID, security.PermAdminUsersMembershipWrite)
}

// CanManageUserSecurity reports whether the user may change status, lock, and verification.
func CanManageUserSecurity(ctx context.Context, userID uint) (bool, error) {
	if ok, err := canAdminUsersFullWrite(ctx, userID); err != nil || ok {
		return ok, err
	}
	return security.Can(ctx, userID, security.PermAdminUsersSecurityWrite)
}

// CanModifyUsersAdmin reports whether the user may POST any user account change in admin.
func CanModifyUsersAdmin(ctx context.Context, userID uint) (bool, error) {
	if ok, err := canAdminUsersFullWrite(ctx, userID); err != nil || ok {
		return ok, err
	}
	for _, perm := range []string{
		security.PermAdminUsersProfileWrite,
		security.PermAdminUsersMembershipWrite,
		security.PermAdminUsersSecurityWrite,
	} {
		ok, err := security.Can(ctx, userID, perm)
		if err != nil {
			return false, err
		}
		if ok {
			return true, nil
		}
	}
	return false, nil
}

// CanCreateAdminUser reports whether the user may create accounts in admin.
func CanCreateAdminUser(ctx context.Context, userID uint) (bool, error) {
	return CanManageUserProfile(ctx, userID)
}

// CanDeleteAdminUser reports whether the user may delete accounts in admin.
func CanDeleteAdminUser(ctx context.Context, userID uint) (bool, error) {
	return canAdminUsersFullWrite(ctx, userID)
}

func userAccessFlags(ctx context.Context) (profile, membership, securityFlag, create, delete bool, err error) {
	svc, err := user.FromContext(ctx)
	if err != nil {
		return false, false, false, false, false, err
	}
	actorID, ok := svc.CurrentID()
	if !ok {
		return false, false, false, false, false, nil
	}
	profile, err = CanManageUserProfile(ctx, actorID)
	if err != nil {
		return false, false, false, false, false, err
	}
	membership, err = CanManageUserMembership(ctx, actorID)
	if err != nil {
		return false, false, false, false, false, err
	}
	securityFlag, err = CanManageUserSecurity(ctx, actorID)
	if err != nil {
		return false, false, false, false, false, err
	}
	create, err = CanCreateAdminUser(ctx, actorID)
	if err != nil {
		return false, false, false, false, false, err
	}
	delete, err = CanDeleteAdminUser(ctx, actorID)
	if err != nil {
		return false, false, false, false, false, err
	}
	return profile, membership, securityFlag, create, delete, nil
}

func validateMembershipAssignment(ctx context.Context, actorID uint, roleIDs, groupIDs []uint) error {
	if actorID == 0 {
		return fmt.Errorf("not signed in")
	}
	super, err := IsSuperAdmin(ctx, actorID)
	if err != nil {
		return err
	}
	if super {
		return nil
	}
	actorRoles, err := security.CollectUserRoleIDs(ctx, actorID)
	if err != nil {
		return err
	}
	roleSet := uintSet(actorRoles)
	for _, id := range roleIDs {
		if _, ok := roleSet[id]; !ok {
			return fmt.Errorf("cannot assign a role you do not hold")
		}
	}
	actorGroups, err := security.CollectUserGroupIDs(ctx, actorID)
	if err != nil {
		return err
	}
	groupSet := uintSet(actorGroups)
	for _, id := range groupIDs {
		if _, ok := groupSet[id]; !ok {
			return fmt.Errorf("cannot assign a group you do not belong to")
		}
	}
	return nil
}

func uintSet(ids []uint) map[uint]struct{} {
	out := make(map[uint]struct{}, len(ids))
	for _, id := range ids {
		if id != 0 {
			out[id] = struct{}{}
		}
	}
	return out
}

func loadUserGroupIDs(db *gorm.DB, userID uint) ([]uint, error) {
	var row models.User
	if err := db.Preload("Groups").First(&row, userID).Error; err != nil {
		return nil, err
	}
	out := make([]uint, 0, len(row.Groups))
	for _, g := range row.Groups {
		out = append(out, g.GroupID)
	}
	return out, nil
}

func filterMembershipOptions(ctx context.Context, actorID uint, allRoles []models.Role, allGroups []models.Group) ([]models.Role, []models.Group, error) {
	super, err := IsSuperAdmin(ctx, actorID)
	if err != nil || super {
		return allRoles, allGroups, nil
	}
	roleSet := uintSet(nil)
	if ids, err := security.CollectUserRoleIDs(ctx, actorID); err != nil {
		return nil, nil, err
	} else {
		roleSet = uintSet(ids)
	}
	groupSet := uintSet(nil)
	if ids, err := security.CollectUserGroupIDs(ctx, actorID); err != nil {
		return nil, nil, err
	} else {
		groupSet = uintSet(ids)
	}
	roles := make([]models.Role, 0, len(allRoles))
	for _, role := range allRoles {
		if _, ok := roleSet[role.RoleID]; ok {
			roles = append(roles, role)
		}
	}
	groups := make([]models.Group, 0, len(allGroups))
	for _, group := range allGroups {
		if _, ok := groupSet[group.GroupID]; ok {
			groups = append(groups, group)
		}
	}
	return roles, groups, nil
}

func loadUserRoleIDs(db *gorm.DB, userID uint) ([]uint, error) {
	var row models.User
	if err := db.Preload("Roles").First(&row, userID).Error; err != nil {
		return nil, err
	}
	out := make([]uint, 0, len(row.Roles))
	for _, r := range row.Roles {
		out = append(out, r.RoleID)
	}
	return out, nil
}
