package security

import (
	"testing"

	"github.com/rob121/cannon/extension"
	"github.com/rob121/cannon/internal/models"
)

func TestGrantExtensionToInstallerAssignsRole(t *testing.T) {
	ctx, db := testContext(t)
	RegisterExtensionPermissions("cannon-ext-demo", []extension.PermissionDef{
		{ID: "manage", DisplayName: "Manage Demo"},
	})
	if err := SyncToDB(db); err != nil {
		t.Fatal(err)
	}

	user := models.User{Username: "installer", Email: "i@example.com", Status: models.StatusActive}
	if err := db.Create(&user).Error; err != nil {
		t.Fatal(err)
	}

	if err := GrantExtensionToInstaller(ctx, db, user.UserID, "cannon-ext-demo"); err != nil {
		t.Fatal(err)
	}
	effective, err := ResolveEffective(ctx, user.UserID)
	if err != nil {
		t.Fatal(err)
	}
	if !HasExtensionAdminAccess(effective, "cannon-ext-demo") {
		t.Fatal("installer should access extension admin")
	}

	var role models.Role
	if err := db.Where("name = ?", extensionInstallerRoleName("cannon-ext-demo")).First(&role).Error; err != nil {
		t.Fatal(err)
	}
	var link int64
	if err := db.Table("user_roles").Where("user_user_id = ? AND role_role_id = ?", user.UserID, role.RoleID).Count(&link).Error; err != nil {
		t.Fatal(err)
	}
	if link != 1 {
		t.Fatalf("expected user linked to installer role, count=%d", link)
	}

	// Idempotent: second grant should not duplicate the assignment.
	if err := GrantExtensionToInstaller(ctx, db, user.UserID, "cannon-ext-demo"); err != nil {
		t.Fatal(err)
	}
	if err := db.Table("user_roles").Where("user_user_id = ? AND role_role_id = ?", user.UserID, role.RoleID).Count(&link).Error; err != nil {
		t.Fatal(err)
	}
	if link != 1 {
		t.Fatalf("expected single role link, count=%d", link)
	}
}

func TestGrantExtensionToInstallerSkipsWhenAlreadyAllowed(t *testing.T) {
	ctx, db := testContext(t)
	RegisterExtensionPermissions("cannon-ext-skip", []extension.PermissionDef{
		{ID: "manage", DisplayName: "Manage"},
	})
	if err := SyncToDB(db); err != nil {
		t.Fatal(err)
	}

	var admin models.Role
	if err := db.Where("name = ?", RoleAdministrator).First(&admin).Error; err != nil {
		t.Fatal(err)
	}
	user := models.User{Username: "admin", Email: "a@example.com", Status: models.StatusActive}
	if err := db.Create(&user).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Model(&user).Association("Roles").Append(&admin); err != nil {
		t.Fatal(err)
	}

	if err := GrantExtensionToInstaller(ctx, db, user.UserID, "cannon-ext-skip"); err != nil {
		t.Fatal(err)
	}
	var count int64
	if err := db.Model(&models.Role{}).Where("name LIKE ?", extensionInstallerRolePrefix+"%").Count(&count).Error; err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatalf("expected no installer role for wildcard admin, got %d roles", count)
	}
}
