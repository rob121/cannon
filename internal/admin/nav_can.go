package admin

import (
	"context"

	"github.com/rob121/cannon/internal/security"
	"github.com/rob121/cannon/internal/user"
)

// navPermissionKeys maps layout nav keys to read permissions.
var navPermissionKeys = map[string]string{
	"dashboard":          security.PermAdminAccess,
	"items":              "core.admin.items.read",
	"review":             "core.admin.items.read",
	"trash":              "core.admin.items.read",
	"categories":         "core.admin.categories.read",
	"tags":               "core.admin.tags.read",
	"field_groups":       "core.admin.field-groups.read",
	"media":              "core.admin.media.read",
	"comments":           "core.admin.comments.read",
	"routes":             "core.admin.routes.read",
	"templates":          "core.admin.templates.read",
	"menus":              "core.admin.menus.read",
	"menu_items":         "core.admin.menu-items.read",
	"accounts":           "core.admin.users.read",
	"authenticators":     "core.admin.authenticators.read",
	"profiles":           "core.admin.profiles.read",
	"groups":             "core.admin.groups.read",
	"roles":              "core.admin.roles.read",
	"permissions":        security.PermPermissionsRead,
	"sites":              "core.admin.sites.read",
	"extension_registry": "core.admin.extensions.read",
	"extension_apps":     security.PermAdminExtensionAppsRead,
	"blocks":             "core.admin.blocks.read",
	"configuration":      "core.admin.configuration.read",
	"notifications":      "core.admin.notifications.read",
	"access_log":         "core.admin.system.read",
	"help":               "core.admin.help.read",
	"languages":          "core.admin.languages.read",
	"api_credentials":    "core.admin.api.read",
	"api_settings":       "core.admin.api.read",
}

func navCanMap(ctx context.Context) map[string]bool {
	out := map[string]bool{}
	svc, err := user.FromContext(ctx)
	if err != nil {
		return out
	}
	userID, ok := svc.CurrentID()
	if !ok {
		return out
	}
	for key, perm := range navPermissionKeys {
		allowed, err := security.Can(ctx, userID, perm)
		out[key] = err == nil && allowed
	}
	return out
}

func navGroupVisible(can map[string]bool, keys ...string) bool {
	for _, key := range keys {
		if can[key] {
			return true
		}
	}
	return false
}
