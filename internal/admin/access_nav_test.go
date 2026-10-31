package admin

import (
	"context"
	"testing"

	"github.com/rob121/cannon/internal/config"
	"github.com/rob121/cannon/internal/database"
	"github.com/rob121/cannon/internal/models"
	"github.com/rob121/cannon/internal/security"
	"github.com/rob121/cannon/internal/sites"
	"gorm.io/gorm"
)

func TestAdminPermissionSectionAliases(t *testing.T) {
	if got := adminPermissionSection("/trash"); got != "/items" {
		t.Fatalf("/trash alias = %q, want /items", got)
	}
	if got := adminPermissionSection("/review"); got != "/items" {
		t.Fatalf("/review alias = %q, want /items", got)
	}
	if adminPermissionSection("/items") != "/items" {
		t.Fatal("expected passthrough for /items")
	}
}

func TestCanAccessAdminRolesReadPermission(t *testing.T) {
	site := &config.SiteConfig{
		ID: "roles-read-access-test",
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
	if err := ensureTestRoleAllows(db, "roles-reader", []string{"core.admin.access", "core.admin.roles.read"}); err != nil {
		t.Fatal(err)
	}
	var role models.Role
	if err := db.Where("name = ?", "roles-reader").First(&role).Error; err != nil {
		t.Fatal(err)
	}
	user := models.User{Username: "rolesreader", Email: "rolesreader@test", Status: models.StatusActive}
	if err := db.Create(&user).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Model(&user).Association("Roles").Append(&role); err != nil {
		t.Fatal(err)
	}
	ctx := sites.WithContext(context.Background(), site)

	ok, err := CanAccessAdmin(ctx, user.UserID, "/admin/roles", false)
	if err != nil || !ok {
		t.Fatalf("roles read access: ok=%v err=%v", ok, err)
	}
	ok, err = CanAccessAdmin(ctx, user.UserID, "/admin/roles", true)
	if err != nil || ok {
		t.Fatalf("roles write should be denied without write permission: ok=%v err=%v", ok, err)
	}
}

func ensureTestRoleAllows(db *gorm.DB, name string, allows []string) error {
	var role models.Role
	err := db.Where("name = ?", name).First(&role).Error
	if err != nil {
		if err != gorm.ErrRecordNotFound {
			return err
		}
		role = models.Role{Name: name, Status: models.StatusActive}
		if err := db.Create(&role).Error; err != nil {
			return err
		}
	}
	return security.SaveRolePermissions(db, role.RoleID, allows, nil)
}
