package security

import (
	"strings"
	"sync"
)

var (
	registryMu sync.RWMutex
	registry   = map[string]Permission{}
	roleDefs   = map[string]RoleDef{}
	policies   = map[string]PolicyDef{}
)

// RegisterPermission registers a permission in the in-memory catalog.
func RegisterPermission(p Permission) {
	if p.ID == "" {
		return
	}
	registryMu.Lock()
	defer registryMu.Unlock()
	registry[p.ID] = p
}

// RegisterRole registers a default role definition in code.
func RegisterRole(def RoleDef) {
	if def.Name == "" {
		return
	}
	registryMu.Lock()
	defer registryMu.Unlock()
	roleDefs[def.Name] = def
}

// RegisterPolicy registers a policy stub for future ABAC support.
func RegisterPolicy(def PolicyDef) {
	if def.ID == "" {
		return
	}
	registryMu.Lock()
	defer registryMu.Unlock()
	policies[def.ID] = def
}

// RegisteredPermissions returns a copy of all registered permissions.
func RegisteredPermissions() []Permission {
	registryMu.RLock()
	defer registryMu.RUnlock()
	out := make([]Permission, 0, len(registry))
	for _, p := range registry {
		out = append(out, p)
	}
	return out
}

// RegisteredRoleDefs returns a copy of all registered role definitions.
func RegisteredRoleDefs() []RoleDef {
	registryMu.RLock()
	defer registryMu.RUnlock()
	out := make([]RoleDef, 0, len(roleDefs))
	for _, r := range roleDefs {
		out = append(out, r)
	}
	return out
}

// PermissionByID returns a registered permission by id.
func PermissionByID(id string) (Permission, bool) {
	registryMu.RLock()
	defer registryMu.RUnlock()
	p, ok := registry[id]
	return p, ok
}

// Categories returns distinct permission categories sorted.
func Categories(perms []Permission) []string {
	seen := map[string]struct{}{}
	var out []string
	for _, p := range perms {
		cat := strings.TrimSpace(p.Category)
		if cat == "" {
			cat = "General"
		}
		if _, ok := seen[cat]; ok {
			continue
		}
		seen[cat] = struct{}{}
		out = append(out, cat)
	}
	return out
}

// PermissionsByCategory groups permissions by category.
func PermissionsByCategory(perms []Permission) map[string][]Permission {
	out := map[string][]Permission{}
	for _, p := range perms {
		cat := strings.TrimSpace(p.Category)
		if cat == "" {
			cat = "General"
		}
		out[cat] = append(out[cat], p)
	}
	return out
}
