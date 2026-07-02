package router_test

import (
	"testing"

	"github.com/rob121/cannon/internal/models"
	"github.com/rob121/cannon/internal/router"
)

func TestBuiltinControllerRoutesPrefixes(t *testing.T) {
	routes := router.BuiltinControllerRoutes()
	if len(routes) == 0 {
		t.Fatal("expected builtin routes")
	}
	seen := map[string]bool{}
	for _, route := range routes {
		seen[route.Controller+"/"+route.ControllerAction] = true
		switch route.ControllerAction {
		case "login", "logout":
			if route.Path != "/auth/"+route.ControllerAction {
				t.Fatalf("auth action %q path: got %q", route.ControllerAction, route.Path)
			}
			if route.Prefix != "auth" {
				t.Fatalf("auth action %q prefix: got %q", route.ControllerAction, route.Prefix)
			}
		case "verify", "verify-resend", "reset-request", "reset-submit":
			if route.Prefix != "account" {
				t.Fatalf("account action %q prefix: got %q", route.ControllerAction, route.Prefix)
			}
		}
	}
	for _, action := range []string{"category", "item", "search"} {
		if !seen["content/"+action] {
			t.Fatalf("missing content/%s route", action)
		}
	}
	if seen["content/index"] {
		t.Fatal("content/index should not be a built-in route")
	}
}

func TestIsBuiltinControllerRoute(t *testing.T) {
	if !router.IsBuiltinControllerRoute(models.Route{Type: models.RouteTypeController, Controller: "content", ControllerAction: "item", Path: "/content/item/*"}) {
		t.Fatal("seeded content item route should be builtin")
	}
	if router.IsBuiltinControllerRoute(models.Route{Type: models.RouteTypeController, Controller: "content", ControllerAction: "index", Path: "/"}) {
		t.Fatal("manual home route should not be builtin")
	}
	if router.IsBuiltinControllerRoute(models.Route{Type: models.RouteTypeURL, Path: "/blog"}) {
		t.Fatal("url route should not be builtin")
	}
}

func TestConflictsWithReservedPath(t *testing.T) {
	if !router.ConflictsWithReservedPath("/content/item/foo") {
		t.Fatal("expected conflict with content item route")
	}
	if !router.ConflictsWithReservedPath("/auth/login") {
		t.Fatal("expected conflict with auth login route")
	}
	if !router.ConflictsWithReservedPath("/admin/users") {
		t.Fatal("expected conflict with admin route")
	}
	if router.ConflictsWithReservedPath("/blog/post") {
		t.Fatal("custom blog path should not conflict")
	}
}
