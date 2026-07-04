package security

// EffectivePermissions holds resolved allow and deny permission sets for a user.
type EffectivePermissions struct {
	Allow map[string]struct{}
	Deny  map[string]struct{}
}

// NewEffectivePermissions returns empty allow/deny sets.
func NewEffectivePermissions() EffectivePermissions {
	return EffectivePermissions{
		Allow: map[string]struct{}{},
		Deny:  map[string]struct{}{},
	}
}

// CheckPermission reports whether a request is allowed after applying explicit denies.
// Explicit deny overrides any matching allow, including wildcards.
func CheckPermission(effective EffectivePermissions, requested string) bool {
	if requested == "" {
		return false
	}
	if HasPermission(effective.Deny, requested) {
		return false
	}
	return HasPermission(effective.Allow, requested)
}

// IsDenied reports whether requested is explicitly denied.
func IsDenied(denied map[string]struct{}, requested string) bool {
	return HasPermission(denied, requested)
}
