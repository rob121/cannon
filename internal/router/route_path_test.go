package router_test

import (
	"testing"

	"github.com/rob121/cannon/internal/paths"
	"github.com/rob121/cannon/internal/router"
)

func TestBuiltinRoutePath(t *testing.T) {
	got := router.BuiltinRoutePath("auth", "login")
	if got != paths.AuthLogin {
		t.Fatalf("got %q want %q", got, paths.AuthLogin)
	}
	if router.BuiltinRoutePath("auth", "missing") != "" {
		t.Fatal("expected empty path for unknown action")
	}
}
