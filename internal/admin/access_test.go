package admin

import (
	"context"
	"testing"

	"github.com/rob121/cannon/extension"
	"github.com/rob121/cannon/internal/config"
	"github.com/rob121/cannon/internal/database"
	"github.com/rob121/cannon/internal/models"
	"github.com/rob121/cannon/internal/roles"
	"github.com/rob121/cannon/internal/security"
	"github.com/rob121/cannon/internal/sites"
)

func TestCanAccessAdminUsesPermissions(t *testing.T) {
	site := &config.SiteConfig{
		ID: "admin-access-test",
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

	var viewer models.Role
	if err := db.Where("name = ?", security.RoleViewer).First(&viewer).Error; err != nil {
		t.Fatal(err)
	}
	user := models.User{Username: "viewer", Email: "viewer@test", Status: models.StatusActive}
	if err := db.Create(&user).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Model(&user).Association("Roles").Append(&viewer); err != nil {
		t.Fatal(err)
	}

	ok, err := CanAccessAdmin(ctx, user.UserID, "/admin/", false)
	if err != nil || !ok {
		t.Fatalf("dashboard access: ok=%v err=%v", ok, err)
	}
	ok, err = CanAccessAdmin(ctx, user.UserID, "/admin/users", false)
	if err != nil || ok {
		t.Fatalf("users read should be denied: ok=%v err=%v", ok, err)
	}

	var admin models.Role
	if err := db.Where("name = ?", security.RoleAdministrator).First(&admin).Error; err != nil {
		t.Fatal(err)
	}
	adminUser := models.User{Username: "admin", Email: "admin@test", Status: models.StatusActive}
	if err := db.Create(&adminUser).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Model(&adminUser).Association("Roles").Append(&admin); err != nil {
		t.Fatal(err)
	}
	ok, err = CanAccessAdmin(ctx, adminUser.UserID, "/admin/users", true)
	if err != nil || !ok {
		t.Fatalf("admin users write: ok=%v err=%v", ok, err)
	}
}

func TestCanAccessExtensionAppsPath(t *testing.T) {
	security.RegisterExtensionPermissions("contact-form", []extension.PermissionDef{
		{ID: "manage", DisplayName: "Manage"},
	})

	site := &config.SiteConfig{
		ID: "ext-apps-access-test",
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

	var viewer models.Role
	if err := db.Where("name = ?", security.RoleViewer).First(&viewer).Error; err != nil {
		t.Fatal(err)
	}
	if err := security.SaveRolePermissions(db, viewer.RoleID, []string{"contact-form.manage"}, nil); err != nil {
		t.Fatal(err)
	}
	user := models.User{Username: "extuser", Email: "extuser@test", Status: models.StatusActive}
	if err := db.Create(&user).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Model(&user).Association("Roles").Append(&viewer); err != nil {
		t.Fatal(err)
	}

	ok, err := CanAccessAdmin(ctx, user.UserID, "/admin/extension-apps/contact-form", false)
	if err != nil || !ok {
		t.Fatalf("extension-specific access: ok=%v err=%v", ok, err)
	}
	ok, err = CanAccessAdmin(ctx, user.UserID, "/admin/extension-apps/other-ext", false)
	if err != nil || ok {
		t.Fatalf("other extension should be denied: ok=%v err=%v", ok, err)
	}

	var editor models.Role
	if err := db.Where("name = ?", security.RoleEditor).First(&editor).Error; err != nil {
		t.Fatal(err)
	}
	editorUser := models.User{Username: "editor", Email: "editor@test", Status: models.StatusActive}
	if err := db.Create(&editorUser).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Model(&editorUser).Association("Roles").Append(&editor); err != nil {
		t.Fatal(err)
	}
	ok, err = CanAccessAdmin(ctx, editorUser.UserID, "/admin/extension-apps/legacy-ext", false)
	if err != nil || !ok {
		t.Fatalf("editor with extension-apps.read should access legacy extension admin: ok=%v err=%v", ok, err)
	}
}
