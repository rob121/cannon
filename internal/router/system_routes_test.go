package router

import "testing"

func TestSystemRoutes(t *testing.T) {
	routes := SystemRoutes()
	if len(routes) < 5 {
		t.Fatalf("expected reserved routes, got %d", len(routes))
	}
	seen := map[string]bool{}
	for _, route := range routes {
		if route.Name == "" || route.Path == "" || route.Handler == "" {
			t.Fatalf("incomplete route: %#v", route)
		}
		if seen[route.Path] {
			t.Fatalf("duplicate path %q", route.Path)
		}
		seen[route.Path] = true
	}
	if !seen["/admin/*"] || !seen["/robots.txt"] {
		t.Fatalf("missing expected reserved paths: %#v", seen)
	}
}
