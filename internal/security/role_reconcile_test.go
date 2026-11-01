package security

import (
	"testing"

	"github.com/rob121/cannon/internal/config"
	"github.com/rob121/cannon/internal/database"
	"github.com/rob121/cannon/internal/models"
)

func TestMigrateSystemRoleDirectPermissionsResetsEditorScope(t *testing.T) {
	site := &config.SiteConfig{
		ID: "role-reconcile-test",
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
	if err := SeedDefaults(db); err != nil {
		t.Fatal(err)
	}

	var editor models.Role
	if err := db.Where("name = ?", RoleEditor).First(&editor).Error; err != nil {
		t.Fatal(err)
	}
	if err := SaveRolePermissions(db, editor.RoleID, []string{
		PermAdminAccess,
		"core.admin.users.read",
		"core.admin.system.read",
	}, nil); err != nil {
		t.Fatal(err)
	}

	if err := migrateSystemRoleDirectPermissions(db); err != nil {
		t.Fatal(err)
	}
	keys, err := LoadRolePermissionKeys(db, editor.RoleID)
	if err != nil {
		t.Fatal(err)
	}
	keySet := map[string]struct{}{}
	for _, k := range keys {
		keySet[k] = struct{}{}
	}
	for _, forbidden := range []string{"core.admin.users.read", "core.admin.system.read", "core.admin.roles.read"} {
		if _, ok := keySet[forbidden]; ok {
			t.Fatalf("editor should not retain %s after reconcile: %#v", forbidden, keys)
		}
	}
	if _, ok := keySet["core.admin.items.write"]; !ok {
		t.Fatalf("editor should have items write: %#v", keys)
	}
}

func TestEnsureAdministratorWildcardGrantRestoresStar(t *testing.T) {
	site := &config.SiteConfig{
		ID: "admin-star-reconcile-test",
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
	if err := SeedDefaults(db); err != nil {
		t.Fatal(err)
	}
	var admin models.Role
	if err := db.Where("name = ?", RoleAdministrator).First(&admin).Error; err != nil {
		t.Fatal(err)
	}
	if err := SaveRolePermissions(db, admin.RoleID, []string{
		"core.admin.items.read",
		"core.roles.manage",
		"core.permissions.read",
	}, nil); err != nil {
		t.Fatal(err)
	}
	if err := saveSecurityMigrationRevision(db, 1); err != nil {
		t.Fatal(err)
	}
	if err := migrateSystemRoleDirectPermissions(db); err != nil {
		t.Fatal(err)
	}
	allows, _, err := LoadRolePermissionAssignments(db, admin.RoleID)
	if err != nil {
		t.Fatal(err)
	}
	hasStar := false
	for _, key := range allows {
		if key == PermWildcardAll {
			hasStar = true
			break
		}
	}
	if !hasStar {
		t.Fatalf("administrator should have * after migration: %#v", allows)
	}
	rev, err := loadSecurityMigrationRevision(db)
	if err != nil || rev < 2 {
		t.Fatalf("migration revision = %d err=%v, want >= 2", rev, err)
	}
}

func TestEnsureForSiteAlwaysRestoresAdministratorWildcard(t *testing.T) {
	site := &config.SiteConfig{
		ID: "ensure-for-site-star-test",
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
	if err := EnsureForSite(db); err != nil {
		t.Fatal(err)
	}
	var admin models.Role
	if err := db.Where("name = ?", RoleAdministrator).First(&admin).Error; err != nil {
		t.Fatal(err)
	}
	if err := SaveRolePermissions(db, admin.RoleID, []string{"core.admin.items.read"}, nil); err != nil {
		t.Fatal(err)
	}
	if err := EnsureForSite(db); err != nil {
		t.Fatal(err)
	}
	allows, _, err := LoadRolePermissionAssignments(db, admin.RoleID)
	if err != nil {
		t.Fatal(err)
	}
	if !containsString(allows, PermWildcardAll) {
		t.Fatalf("administrator should have * after EnsureForSite: %#v", allows)
	}
}

func TestWithAdministratorWildcardAllow(t *testing.T) {
	out := WithAdministratorWildcardAllow([]string{"core.admin.items.read"})
	if !containsString(out, PermWildcardAll) {
		t.Fatalf("expected * in %#v", out)
	}
	dup := WithAdministratorWildcardAllow(out)
	if len(dup) != len(out) {
		t.Fatalf("should not duplicate *: %#v", dup)
	}
}

func containsString(list []string, want string) bool {
	for _, s := range list {
		if s == want {
			return true
		}
	}
	return false
}
