package admin

import (
	"context"
	"testing"

	"github.com/rob121/cannon/internal/config"
	"github.com/rob121/cannon/internal/database"
	"github.com/rob121/cannon/internal/models"
	"github.com/rob121/cannon/internal/roles"
	"github.com/rob121/cannon/internal/security"
	"github.com/rob121/cannon/internal/sites"
)

func TestValidateMembershipAssignmentRejectsEscalation(t *testing.T) {
	site := &config.SiteConfig{
		ID: "user-access-test",
		Database: config.DatabaseConfig{
			Type: "sqlite",
			DSN:  t.TempDir() + "/site.db",
		},
	}
	if err := database.Migrate(site); err != nil {
		t.Fatal(err)
	}
	db, err := database.Get(site.ID)
	if err != nil {
		t.Fatal(err)
	}
	if err := roles.EnsureDefaults(db); err != nil {
		t.Fatal(err)
	}
	ctx := sites.WithContext(context.Background(), site)

	var editor, administrator models.Role
	if err := db.Where("name = ?", security.RoleEditor).First(&editor).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Where("name = ?", security.RoleAdministrator).First(&administrator).Error; err != nil {
		t.Fatal(err)
	}

	actor := models.User{Username: "editor", Email: "editor@test", Status: models.StatusActive}
	if err := db.Create(&actor).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Model(&actor).Association("Roles").Append(&editor); err != nil {
		t.Fatal(err)
	}

	if err := validateMembershipAssignment(ctx, actor.UserID, []uint{administrator.RoleID}, nil); err == nil {
		t.Fatal("expected assigning administrator role to fail for editor actor")
	}
	if err := validateMembershipAssignment(ctx, actor.UserID, []uint{editor.RoleID}, nil); err != nil {
		t.Fatalf("editor assigning editor role: %v", err)
	}
}

func TestGranularUserWritePermissions(t *testing.T) {
	site := &config.SiteConfig{
		ID: "user-granular-perm-test",
		Database: config.DatabaseConfig{
			Type: "sqlite",
			DSN:  t.TempDir() + "/site.db",
		},
	}
	if err := database.Migrate(site); err != nil {
		t.Fatal(err)
	}
	db, err := database.Get(site.ID)
	if err != nil {
		t.Fatal(err)
	}
	if err := roles.EnsureDefaults(db); err != nil {
		t.Fatal(err)
	}
	ctx := sites.WithContext(context.Background(), site)

	user := models.User{Username: "profile", Email: "profile@test", Status: models.StatusActive}
	if err := db.Create(&user).Error; err != nil {
		t.Fatal(err)
	}
	role := models.Role{Name: "profile_manager", Status: models.StatusActive}
	if err := db.Create(&role).Error; err != nil {
		t.Fatal(err)
	}
	if err := security.SaveRolePermissions(db, role.RoleID, []string{
		security.PermAdminAccess,
		security.AdminPermissionForPath("/users", false),
		security.PermAdminUsersProfileWrite,
	}, nil); err != nil {
		t.Fatal(err)
	}
	if err := db.Model(&user).Association("Roles").Append(&role); err != nil {
		t.Fatal(err)
	}
	security.InvalidateSiteUser(ctx, user.UserID)

	ok, err := CanManageUserProfile(ctx, user.UserID)
	if err != nil || !ok {
		t.Fatalf("profile write: ok=%v err=%v", ok, err)
	}
	ok, err = CanManageUserMembership(ctx, user.UserID)
	if err != nil || ok {
		t.Fatalf("membership should be denied: ok=%v err=%v", ok, err)
	}
	ok, err = CanModifyUsersAdmin(ctx, user.UserID)
	if err != nil || !ok {
		t.Fatalf("should allow POST with profile perm: ok=%v err=%v", ok, err)
	}
}
