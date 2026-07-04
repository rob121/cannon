package extension_test

import (
	"testing"

	"github.com/rob121/cannon/extension"
)

func TestUserCanExactAndWildcard(t *testing.T) {
	req := extension.WireRequest{
		User: map[string]any{
			"permissions": []any{"myext.manage", "blog.*"},
		},
	}
	if !extension.UserCan(req, "myext.manage") {
		t.Fatal("expected exact match")
	}
	if !extension.UserCan(req, "blog.article.publish") {
		t.Fatal("expected wildcard prefix match")
	}
	if extension.UserCan(req, "other.read") {
		t.Fatal("expected deny")
	}
}

func TestUserCanAdministratorWildcard(t *testing.T) {
	req := extension.WireRequest{
		User: map[string]any{
			"permissions": []any{"*"},
		},
	}
	if !extension.UserCan(req, "myext.anything") {
		t.Fatal("expected * to grant extension permission")
	}
}

func TestUserCanExplicitDeny(t *testing.T) {
	req := extension.WireRequest{
		User: map[string]any{
			"permissions":        []any{"*"},
			"denied_permissions": []any{"myext.manage"},
		},
	}
	if extension.UserCan(req, "myext.manage") {
		t.Fatal("expected explicit deny to override wildcard allow")
	}
	if !extension.UserCan(req, "myext.read") {
		t.Fatal("expected non-denied permission to remain allowed")
	}
}

func TestUserCanMissingPermissions(t *testing.T) {
	req := extension.WireRequest{User: map[string]any{"authenticated": true}}
	if extension.UserCan(req, "myext.manage") {
		t.Fatal("expected deny without permissions array")
	}
}
