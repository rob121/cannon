package extension

import "strings"

// PermissionDef describes one extension permission for registration with Cannon.
type PermissionDef struct {
	ID          string `json:"id"`
	DisplayName string `json:"display_name"`
	Description string `json:"description,omitempty"`
	Category    string `json:"category,omitempty"`
	Dangerous   bool   `json:"dangerous,omitempty"`
}

// UserCan reports whether the wire user scope includes a permission.
// Supports wildcard grants such as *, core.*, and myext.*.
// Explicit denies in denied_permissions override allows.
func UserCan(req WireRequest, permissionID string) bool {
	if permissionID == "" {
		return false
	}
	if denied, ok := req.User["denied_permissions"].([]any); ok && permissionDenied(denied, permissionID) {
		return false
	}
	perms, ok := req.User["permissions"].([]any)
	if !ok {
		return false
	}
	granted := make([]string, 0, len(perms))
	for _, item := range perms {
		if s, ok := item.(string); ok && s != "" {
			granted = append(granted, s)
		}
	}
	return permissionAllowed(granted, permissionID)
}

func permissionDenied(denied []any, permissionID string) bool {
	keys := make([]string, 0, len(denied))
	for _, item := range denied {
		if s, ok := item.(string); ok && s != "" {
			keys = append(keys, s)
		}
	}
	return permissionAllowed(keys, permissionID)
}

func permissionAllowed(granted []string, requested string) bool {
	for _, g := range granted {
		if permissionMatches(g, requested) {
			return true
		}
	}
	for _, candidate := range permissionCandidates(requested) {
		for _, g := range granted {
			if g == candidate {
				return true
			}
		}
	}
	return false
}

func permissionMatches(pattern, requested string) bool {
	if pattern == "" || requested == "" {
		return false
	}
	if pattern == "*" || pattern == requested {
		return true
	}
	if strings.HasSuffix(pattern, ".*") {
		prefix := strings.TrimSuffix(pattern, ".*")
		return requested == prefix || strings.HasPrefix(requested, prefix+".")
	}
	if strings.HasPrefix(pattern, "*.") {
		suffix := strings.TrimPrefix(pattern, "*.")
		return requested == suffix || strings.HasSuffix(requested, "."+suffix)
	}
	return false
}

func permissionCandidates(permissionID string) []string {
	if permissionID == "" {
		return nil
	}
	parts := strings.Split(permissionID, ".")
	out := []string{permissionID, "*"}
	for i := len(parts) - 1; i > 0; i-- {
		out = append(out, strings.Join(parts[:i], ".")+".*")
	}
	return out
}
