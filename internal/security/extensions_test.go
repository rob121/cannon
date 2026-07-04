package security

import (
	"testing"

	"github.com/rob121/cannon/extension"
)

func TestHasExtensionAdminAccess(t *testing.T) {
	RegisterExtensionPermissions("contact-form", []extension.PermissionDef{
		{ID: "manage", DisplayName: "Manage"},
		{ID: "settings", DisplayName: "Settings"},
	})

	effective := EffectivePermissions{
		Allow: map[string]struct{}{"contact-form.manage": {}},
	}
	if !HasExtensionAdminAccess(effective, "contact-form") {
		t.Fatal("expected specific extension permission to grant access")
	}
	if HasExtensionAdminAccess(effective, "calendar") {
		t.Fatal("expected deny for other extension")
	}

	effective = EffectivePermissions{
		Allow: map[string]struct{}{PermAdminExtensionAppsRead: {}},
	}
	if HasExtensionAdminAccess(effective, "contact-form") {
		t.Fatal("extension-apps.read should not bypass registered extension permissions")
	}

	effective = EffectivePermissions{
		Allow: map[string]struct{}{"contact-form.*": {}},
	}
	if !HasExtensionAdminAccess(effective, "contact-form") {
		t.Fatal("expected extension wildcard to grant access")
	}

	effective = EffectivePermissions{
		Allow: map[string]struct{}{"*": {}},
	}
	if !HasExtensionAdminAccess(effective, "contact-form") {
		t.Fatal("expected global wildcard to grant access")
	}
}

func TestHasExtensionAdminAccessWithDeny(t *testing.T) {
	RegisterExtensionPermissions("contact-form", []extension.PermissionDef{
		{ID: "manage", DisplayName: "Manage"},
	})
	effective := EffectivePermissions{
		Allow: map[string]struct{}{"contact-form.manage": {}, PermAdminExtensionAppsRead: {}},
		Deny:  map[string]struct{}{"contact-form.manage": {}},
	}
	if HasExtensionAdminAccess(effective, "contact-form") {
		t.Fatal("expected explicit deny to block extension admin access")
	}
}

func TestHasExtensionAdminAccessWithoutRegisteredPerms(t *testing.T) {
	effective := EffectivePermissions{
		Allow: map[string]struct{}{PermAdminExtensionAppsRead: {}},
	}
	if !HasExtensionAdminAccess(effective, "legacy-extension") {
		t.Fatal("expected extension-apps.read for extension without registered permissions")
	}
	effective = EffectivePermissions{
		Allow: map[string]struct{}{"core.admin.items.read": {}},
	}
	if HasExtensionAdminAccess(effective, "legacy-extension") {
		t.Fatal("expected deny without extension-apps.read or extension permission")
	}
}
