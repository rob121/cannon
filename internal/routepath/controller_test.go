package routepath_test

import (
	"context"
	"testing"

	"github.com/rob121/cannon/internal/paths"
	"github.com/rob121/cannon/internal/routepath"
)

func TestControllerFallsBackToBuiltin(t *testing.T) {
	got := routepath.Controller(context.Background(), "auth", "login")
	if got != paths.AuthLogin {
		t.Fatalf("got %q want %q", got, paths.AuthLogin)
	}
}

func TestControllerWithSuffix(t *testing.T) {
	got := routepath.ControllerWithSuffix(context.Background(), "auth", "reset-submit", "abc123")
	want := "/account/reset-password/abc123"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}
