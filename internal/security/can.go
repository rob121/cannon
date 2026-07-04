package security

import (
	"context"

	"github.com/rob121/cannon/internal/sites"
)

type contextKey struct{}

// WithPermissions attaches a resolved permission set to context.
func WithPermissions(ctx context.Context, perms EffectivePermissions) context.Context {
	return context.WithValue(ctx, contextKey{}, cloneEffective(perms))
}

// PermissionsFromContext returns request-scoped permissions when present.
func PermissionsFromContext(ctx context.Context) (EffectivePermissions, bool) {
	perms, ok := ctx.Value(contextKey{}).(EffectivePermissions)
	if !ok {
		return EffectivePermissions{}, false
	}
	return cloneEffective(perms), true
}

// Can reports whether userID may perform the requested permission. Defaults to deny.
func Can(ctx context.Context, userID uint, permissionID string) (bool, error) {
	if permissionID == "" || userID == 0 {
		return false, nil
	}
	perms, ok := PermissionsFromContext(ctx)
	if !ok {
		var err error
		perms, err = ResolveEffective(ctx, userID)
		if err != nil {
			return false, err
		}
	}
	return CheckPermission(perms, permissionID), nil
}

// CanCurrent checks a permission for the request-scoped user cache.
func CanCurrent(ctx context.Context, permissionID string) (bool, error) {
	perms, ok := PermissionsFromContext(ctx)
	if !ok {
		return false, nil
	}
	return CheckPermission(perms, permissionID), nil
}

// PreloadForUser resolves and attaches permissions to context for an authenticated user.
func PreloadForUser(ctx context.Context, userID uint) (context.Context, error) {
	perms, err := ResolveEffective(ctx, userID)
	if err != nil {
		return ctx, err
	}
	return WithPermissions(ctx, perms), nil
}

// PermissionKeys returns sorted allow permission keys for templates/extensions.
func PermissionKeys(perms EffectivePermissions) []string {
	return permissionKeysFromSet(perms.Allow)
}

// DeniedPermissionKeys returns sorted deny permission keys for templates/extensions.
func DeniedPermissionKeys(perms EffectivePermissions) []string {
	return permissionKeysFromSet(perms.Deny)
}

func permissionKeysFromSet(set map[string]struct{}) []string {
	if len(set) == 0 {
		return nil
	}
	out := make([]string, 0, len(set))
	for k := range set {
		out = append(out, k)
	}
	return out
}

// InvalidateSiteUser clears cache after assignment changes.
func InvalidateSiteUser(ctx context.Context, userID uint) {
	site, err := sites.FromContext(ctx)
	if err != nil {
		InvalidateAll()
		return
	}
	InvalidateUser(site.ID, userID)
}

// InvalidateSiteContext clears all users for the current site.
func InvalidateSiteContext(ctx context.Context) {
	site, err := sites.FromContext(ctx)
	if err != nil {
		InvalidateAll()
		return
	}
	InvalidateSite(site.ID)
}
