package security

import (
	"context"

	"github.com/rob121/cannon/extension"
)

var extensionAdminPermissions = map[string][]string{}

// RegisterExtensionPermissions registers extension-owned permissions with a namespace prefix.
func RegisterExtensionPermissions(extensionName string, perms []extension.PermissionDef) {
	prefix := extensionName + "."
	ids := make([]string, 0, len(perms))
	for _, p := range perms {
		id := p.ID
		if id == "" {
			continue
		}
		if id != "*" && !hasPrefix(id, prefix) {
			id = prefix + id
		}
		ids = append(ids, id)
		category := p.Category
		if category == "" {
			category = extensionName
		}
		RegisterPermission(Permission{
			ID:          id,
			DisplayName: p.DisplayName,
			Description: p.Description,
			Category:    category,
			Dangerous:   p.Dangerous,
		})
	}
	extensionAdminPermissions[extensionName] = ids
}

// ExtensionAdminPermissionIDs returns registered permission ids for an extension admin UI.
func ExtensionAdminPermissionIDs(extensionName string) []string {
	return append([]string(nil), extensionAdminPermissions[extensionName]...)
}

// CanAccessExtensionAdmin reports whether a user may open an extension admin UI.
func CanAccessExtensionAdmin(ctx context.Context, userID uint, extensionName string) (bool, error) {
	perms, err := ResolveEffective(ctx, userID)
	if err != nil {
		return false, err
	}
	return HasExtensionAdminAccess(perms, extensionName), nil
}

// HasExtensionAdminAccess checks a resolved permission set for extension admin access.
func HasExtensionAdminAccess(effective EffectivePermissions, extensionName string) bool {
	if CheckPermission(effective, PermWildcardAll) {
		return true
	}
	permIDs := extensionAdminPermissions[extensionName]
	if len(permIDs) > 0 {
		for _, id := range permIDs {
			if CheckPermission(effective, id) {
				return true
			}
		}
		prefix := extensionName + "."
		if CheckPermission(effective, prefix+"*") {
			return true
		}
		return false
	}
	return CheckPermission(effective, PermAdminExtensionAppsRead)
}

func hasPrefix(s, prefix string) bool {
	return len(s) >= len(prefix) && s[:len(prefix)] == prefix
}
