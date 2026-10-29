package security

import (
	"context"
	"strings"
	"sync"

	"github.com/rob121/cannon/extension"
)

var (
	extensionPermMu           sync.Mutex
	extensionAdminPermissions = map[string][]string{}
)

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
	extensionPermMu.Lock()
	extensionAdminPermissions[extensionName] = ids
	extensionPermMu.Unlock()
}

// UnregisterExtensionPermissions removes an extension's permissions from the in-memory catalog.
func UnregisterExtensionPermissions(extensionName string) {
	extensionName = strings.TrimSpace(extensionName)
	if extensionName == "" {
		return
	}
	extensionPermMu.Lock()
	ids := append([]string(nil), extensionAdminPermissions[extensionName]...)
	delete(extensionAdminPermissions, extensionName)
	extensionPermMu.Unlock()

	registryMu.Lock()
	defer registryMu.Unlock()
	for _, id := range ids {
		delete(registry, id)
	}
}

// ExtensionAdminPermissionIDs returns registered permission ids for an extension admin UI.
func ExtensionAdminPermissionIDs(extensionName string) []string {
	extensionPermMu.Lock()
	defer extensionPermMu.Unlock()
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
	extensionPermMu.Lock()
	permIDs := append([]string(nil), extensionAdminPermissions[extensionName]...)
	extensionPermMu.Unlock()
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
