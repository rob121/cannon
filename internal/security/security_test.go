package security

import (
	"context"
	"testing"

	"github.com/rob121/cannon/internal/config"
	"github.com/rob121/cannon/internal/database"
	"github.com/rob121/cannon/internal/models"
	"github.com/rob121/cannon/internal/sites"
	"gorm.io/gorm"
)

func testContext(t *testing.T) (context.Context, *gorm.DB) {
	t.Helper()
	site := &config.SiteConfig{
		ID: "sec-test",
		Database: config.DatabaseConfig{
			Type: "sqlite",
			DSN:  t.TempDir() + "/site.db",
		},
	}
	db, err := database.Connect(site)
	if err != nil {
		t.Fatal(err)
	}
	for _, model := range models.All() {
		if err := db.AutoMigrate(model); err != nil {
			t.Fatal(err)
		}
	}
	if err := EnsureForSite(db); err != nil {
		t.Fatal(err)
	}
	ctx := sites.WithContext(context.Background(), site)
	return ctx, db
}

func TestWildcardMatching(t *testing.T) {
	granted := map[string]struct{}{
		"blog.*": {},
	}
	if !HasPermission(granted, "blog.article.update") {
		t.Fatal("expected blog.* to grant blog.article.update")
	}
	if HasPermission(granted, "core.admin.access") {
		t.Fatal("blog.* should not grant core permissions")
	}
	granted = map[string]struct{}{"*": {}}
	if !HasPermission(granted, "anything.here") {
		t.Fatal("expected * to grant any permission")
	}
}

func TestRoleInheritanceResolution(t *testing.T) {
	ctx, db := testContext(t)

	var viewer, editor models.Role
	if err := db.Where("name = ?", RoleViewer).First(&viewer).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Where("name = ?", RoleEditor).First(&editor).Error; err != nil {
		t.Fatal(err)
	}
	if err := SaveRolePermissions(db, viewer.RoleID, []string{"core.view.test"}, nil); err != nil {
		t.Fatal(err)
	}
	var writer models.Role
	if err := db.Where("name = ?", RoleWriter).First(&writer).Error; err != nil {
		t.Fatal(err)
	}
	if err := SaveRoleInheritance(db, writer.RoleID, []uint{viewer.RoleID}); err != nil {
		t.Fatal(err)
	}
	if err := SaveRoleInheritance(db, editor.RoleID, []uint{writer.RoleID}); err != nil {
		t.Fatal(err)
	}

	user := models.User{Username: "editor", Email: "editor@test", Status: models.StatusActive}
	if err := db.Create(&user).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Model(&user).Association("Roles").Append(&editor); err != nil {
		t.Fatal(err)
	}

	perms, err := ResolveEffective(ctx, user.UserID)
	if err != nil {
		t.Fatal(err)
	}
	if !HasPermission(perms.Allow, "core.view.test") {
		t.Fatal("editor should inherit viewer permissions")
	}
}

func TestCanDefaultsToDeny(t *testing.T) {
	ctx, db := testContext(t)
	user := models.User{Username: "nobody", Email: "nobody@test", Status: models.StatusActive}
	if err := db.Create(&user).Error; err != nil {
		t.Fatal(err)
	}
	ok, err := Can(ctx, user.UserID, PermContentFrontendCreate)
	if err != nil {
		t.Fatal(err)
	}
	if ok {
		t.Fatal("expected deny by default")
	}
}

func TestAdministratorWildcard(t *testing.T) {
	ctx, db := testContext(t)
	var adminRole models.Role
	if err := db.Where("name = ?", RoleAdministrator).First(&adminRole).Error; err != nil {
		t.Fatal(err)
	}
	user := models.User{Username: "admin", Email: "admin@test", Status: models.StatusActive}
	if err := db.Create(&user).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Model(&user).Association("Roles").Append(&adminRole); err != nil {
		t.Fatal(err)
	}
	ok, err := Can(ctx, user.UserID, "core.admin.users.write")
	if err != nil || !ok {
		t.Fatalf("admin wildcard: ok=%v err=%v", ok, err)
	}
}

func TestCacheInvalidation(t *testing.T) {
	ctx, db := testContext(t)
	var writer models.Role
	if err := db.Where("name = ?", RoleWriter).First(&writer).Error; err != nil {
		t.Fatal(err)
	}
	user := models.User{Username: "writer", Email: "writer@test", Status: models.StatusActive}
	if err := db.Create(&user).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Model(&user).Association("Roles").Append(&writer); err != nil {
		t.Fatal(err)
	}
	if _, err := ResolveEffective(ctx, user.UserID); err != nil {
		t.Fatal(err)
	}
	site, _ := sites.FromContext(ctx)
	InvalidateUser(site.ID, user.UserID)
	if err := SaveRolePermissions(db, writer.RoleID, []string{PermContentFrontendDelete}, nil); err != nil {
		t.Fatal(err)
	}
	ok, err := Can(ctx, user.UserID, PermContentFrontendDelete)
	if err != nil || !ok {
		t.Fatalf("expected refreshed permissions: ok=%v err=%v", ok, err)
	}
}

func TestValidateInheritanceCycle(t *testing.T) {
	_, db := testContext(t)
	var viewer, editor, manager models.Role
	for _, name := range []string{RoleViewer, RoleEditor, RoleManager} {
		var role models.Role
		if err := db.Where("name = ?", name).First(&role).Error; err != nil {
			t.Fatal(err)
		}
		switch name {
		case RoleViewer:
			viewer = role
		case RoleEditor:
			editor = role
		case RoleManager:
			manager = role
		}
	}
	if err := SaveRoleInheritance(db, editor.RoleID, []uint{viewer.RoleID}); err != nil {
		t.Fatal(err)
	}
	if err := SaveRoleInheritance(db, manager.RoleID, []uint{editor.RoleID}); err != nil {
		t.Fatal(err)
	}
	if err := ValidateInheritance(db, viewer.RoleID, []uint{manager.RoleID}); err == nil {
		t.Fatal("expected cycle detection")
	}
}

func TestMigrateContentPermissionKeys(t *testing.T) {
	_, db := testContext(t)
	var writer models.Role
	if err := db.Where("name = ?", RoleWriter).First(&writer).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Where("role_id = ?", writer.RoleID).Delete(&models.RolePermission{}).Error; err != nil {
		t.Fatal(err)
	}
	legacy := "core.content.edit"
	if err := db.Create(&models.RolePermission{RoleID: writer.RoleID, PermissionKey: legacy}).Error; err != nil {
		t.Fatal(err)
	}
	if err := migrateContentPermissionKeys(db); err != nil {
		t.Fatal(err)
	}
	keys, err := LoadRolePermissionKeys(db, writer.RoleID)
	if err != nil {
		t.Fatal(err)
	}
	if len(keys) != 1 || keys[0] != PermContentFrontendEdit {
		t.Fatalf("expected migrated key %q, got %v", PermContentFrontendEdit, keys)
	}
}

func TestDeprecatedPermissionStillHonored(t *testing.T) {
	ctx, db := testContext(t)
	legacy := "core.content.edit"
	if err := db.Create(&models.Permission{
		Key:         legacy,
		DisplayName: "Legacy edit",
		Category:    "content",
		Deprecated:  true,
	}).Error; err != nil {
		t.Fatal(err)
	}
	var viewer models.Role
	if err := db.Where("name = ?", RoleViewer).First(&viewer).Error; err != nil {
		t.Fatal(err)
	}
	if err := SaveRolePermissions(db, viewer.RoleID, []string{legacy}, nil); err != nil {
		t.Fatal(err)
	}
	user := models.User{Username: "legacy", Email: "legacy@test", Status: models.StatusActive}
	if err := db.Create(&user).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Model(&user).Association("Roles").Append(&viewer); err != nil {
		t.Fatal(err)
	}
	ok, err := Can(ctx, user.UserID, legacy)
	if err != nil || !ok {
		t.Fatalf("deprecated assigned permission should still grant: ok=%v err=%v", ok, err)
	}
}

func TestResolveEffectivePopulatesCache(t *testing.T) {
	ctx, db := testContext(t)
	var writer models.Role
	if err := db.Where("name = ?", RoleWriter).First(&writer).Error; err != nil {
		t.Fatal(err)
	}
	user := models.User{Username: "cached", Email: "cached@test", Status: models.StatusActive}
	if err := db.Create(&user).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Model(&user).Association("Roles").Append(&writer); err != nil {
		t.Fatal(err)
	}
	site, _ := sites.FromContext(ctx)
	if _, ok := cacheGet(site.ID, user.UserID); ok {
		t.Fatal("cache should be empty before resolve")
	}
	if _, err := ResolveEffective(ctx, user.UserID); err != nil {
		t.Fatal(err)
	}
	if _, ok := cacheGet(site.ID, user.UserID); !ok {
		t.Fatal("expected cache populated after first resolve")
	}
}
