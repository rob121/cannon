package security

import (
	"testing"

	"github.com/rob121/cannon/internal/models"
)

func TestExplicitDenyOverridesAllow(t *testing.T) {
	effective := EffectivePermissions{
		Allow: map[string]struct{}{"*": {}},
		Deny:  map[string]struct{}{"core.admin.users.write": {}},
	}
	if CheckPermission(effective, "core.admin.users.write") {
		t.Fatal("expected explicit deny to override wildcard allow")
	}
	if !CheckPermission(effective, "core.admin.items.read") {
		t.Fatal("expected other permissions to remain allowed")
	}
}

func TestExplicitDenyWildcard(t *testing.T) {
	effective := EffectivePermissions{
		Allow: map[string]struct{}{"core.admin.*": {}},
		Deny:  map[string]struct{}{"core.admin.users.*": {}},
	}
	if CheckPermission(effective, "core.admin.users.read") {
		t.Fatal("expected wildcard deny to block users admin permissions")
	}
	if !CheckPermission(effective, "core.admin.items.read") {
		t.Fatal("expected non-denied admin permissions to remain allowed")
	}
}

func TestExplicitDenyResolvedFromRoles(t *testing.T) {
	ctx, db := testContext(t)
	var viewer, editor models.Role
	if err := db.Where("name = ?", RoleViewer).First(&viewer).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Where("name = ?", RoleEditor).First(&editor).Error; err != nil {
		t.Fatal(err)
	}
	if err := SaveRolePermissions(db, editor.RoleID, []string{PermAdminAccess}, nil); err != nil {
		t.Fatal(err)
	}
	if err := SaveRolePermissions(db, viewer.RoleID, nil, []string{PermAdminAccess}); err != nil {
		t.Fatal(err)
	}

	user := models.User{Username: "limited", Email: "limited@test", Status: models.StatusActive}
	if err := db.Create(&user).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Model(&user).Association("Roles").Append(&editor, &viewer); err != nil {
		t.Fatal(err)
	}

	ok, err := Can(ctx, user.UserID, PermAdminAccess)
	if err != nil || ok {
		t.Fatalf("expected deny from viewer role to override editor allow: ok=%v err=%v", ok, err)
	}
}

func TestSaveRolePermissionsRejectsOverlap(t *testing.T) {
	_, db := testContext(t)
	var viewer models.Role
	if err := db.Where("name = ?", RoleViewer).First(&viewer).Error; err != nil {
		t.Fatal(err)
	}
	err := SaveRolePermissions(db, viewer.RoleID, []string{PermAdminAccess}, []string{PermAdminAccess})
	if err == nil {
		t.Fatal("expected error when permission is both allowed and denied on same role")
	}
}
