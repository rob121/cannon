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
