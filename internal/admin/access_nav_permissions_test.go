package admin

import (
	"testing"

	"github.com/rob121/cannon/internal/security"
)

// navPathForKey maps layout nav keys to a representative admin URL path.
var navPathForKey = map[string]string{
	"dashboard":          "/admin/",
	"items":              "/admin/items",
	"review":             "/admin/review",
	"trash":              "/admin/trash",
	"categories":         "/admin/categories",
	"tags":               "/admin/tags",
	"field_groups":       "/admin/field-groups",
	"media":              "/admin/media",
	"comments":           "/admin/comments",
	"routes":             "/admin/routes",
	"templates":          "/admin/templates",
	"menus":              "/admin/menus",
	"menu_items":         "/admin/menu-items",
	"accounts":           "/admin/users",
	"authenticators":     "/admin/authenticators",
	"profiles":           "/admin/profiles",
	"groups":             "/admin/groups",
	"roles":              "/admin/roles",
	"permissions":        "/admin/permissions",
	"sites":              "/admin/sites",
	"extension_registry": "/admin/extensions",
	"extension_apps":     "/admin/extension-apps",
	"blocks":             "/admin/blocks",
	"configuration":      "/admin/configuration",
	"notifications":      "/admin/notifications",
	"access_log":         "/admin/system",
	"system_update":      "/admin/system",
	"help":               "/admin/help",
	"languages":          "/admin/languages",
	"api_credentials":    "/admin/api",
	"api_settings":       "/admin/api",
}

func TestNavPermissionKeysMatchAdminPaths(t *testing.T) {
	for navKey, perm := range navPermissionKeys {
		path, ok := navPathForKey[navKey]
		if !ok {
			t.Fatalf("nav key %q missing navPathForKey entry", navKey)
		}
		section := adminSection(path)
		aliasSection := adminPermissionSection(section)
		mapped := security.AdminPermissionForPath(aliasSection, false)
		if mapped != perm && perm != security.PermAdminAccess && perm != security.PermPermissionsRead && perm != security.PermAdminExtensionAppsRead {
			// Nav uses explicit permission IDs; they must match the read permission for the section.
			if mapped != perm {
				t.Fatalf("nav %q perm %q != AdminPermissionForPath(%q)=%q", navKey, perm, aliasSection, mapped)
			}
		}
	}
}

func TestTrashReviewUseItemsSection(t *testing.T) {
	for _, path := range []string{"/admin/trash", "/admin/review"} {
		section := adminPermissionSection(adminSection(path))
		if section != "/items" {
			t.Fatalf("%s section = %q", path, section)
		}
	}
}
