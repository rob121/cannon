package security

import "strings"

// Matches reports whether pattern grants access to requested.
// Supports *, prefix.*, and *.suffix patterns.
func Matches(pattern, requested string) bool {
	if pattern == "" || requested == "" {
		return false
	}
	if pattern == "*" {
		return true
	}
	if pattern == requested {
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

// CandidatePatterns returns wildcard patterns implied by a permission id.
func CandidatePatterns(permissionID string) []string {
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

// HasPermission checks granted permissions against a requested permission id.
func HasPermission(granted map[string]struct{}, requested string) bool {
	if _, ok := granted[requested]; ok {
		return true
	}
	for g := range granted {
		if Matches(g, requested) {
			return true
		}
	}
	for _, candidate := range CandidatePatterns(requested) {
		if _, ok := granted[candidate]; ok {
			return true
		}
	}
	return false
}
