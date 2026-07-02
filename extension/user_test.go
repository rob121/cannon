package extension_test

import (
	"testing"

	"github.com/rob121/cannon/extension"
)

func TestUserHelpers(t *testing.T) {
	req := extension.WireRequest{
		User: map[string]any{
			"authenticated": true,
			"user_id":         float64(42),
			"username":        "jane",
			"email":           "jane@example.com",
			"given_name":      "Jane",
			"family_name":     "Doe",
		},
	}
	if !extension.UserAuthenticated(req) {
		t.Fatal("expected authenticated")
	}
	if id, ok := extension.UserID(req); !ok || id != 42 {
		t.Fatalf("user id: got %d ok=%v", id, ok)
	}
	if got := extension.UserDisplayName(req); got != "Jane Doe" {
		t.Fatalf("display name: got %q", got)
	}
	if got := extension.UserString(req, "email"); got != "jane@example.com" {
		t.Fatalf("email: got %q", got)
	}
}

func TestUserDisplayNameFallbacks(t *testing.T) {
	req := extension.WireRequest{
		User: map[string]any{
			"authenticated": true,
			"username":        "jane",
		},
	}
	if got := extension.UserDisplayName(req); got != "jane" {
		t.Fatalf("username fallback: got %q", got)
	}

	req.User = map[string]any{
		"authenticated": true,
		"email":           "jane@example.com",
	}
	if got := extension.UserDisplayName(req); got != "jane@example.com" {
		t.Fatalf("email fallback: got %q", got)
	}
}

func TestUserUnauthenticated(t *testing.T) {
	req := extension.WireRequest{
		User: map[string]any{"authenticated": false},
	}
	if extension.UserAuthenticated(req) {
		t.Fatal("expected unauthenticated")
	}
	if extension.UserDisplayName(req) != "" {
		t.Fatal("expected empty display name")
	}
}
