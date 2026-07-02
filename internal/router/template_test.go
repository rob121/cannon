package router_test

import (
	"context"
	"testing"

	"github.com/rob121/cannon/internal/models"
	"github.com/rob121/cannon/internal/router"
)

func TestIsDefaultRouteContext(t *testing.T) {
	home := models.Route{RouteID: 1, Name: "Home", Path: "/", IsDefault: true}
	blog := models.Route{RouteID: 2, Name: "Blog", Path: "/blog", IsDefault: false}

	if router.IsDefaultRouteContext(context.Background()) {
		t.Fatal("expected false without matched route")
	}
	if !router.IsDefaultRouteContext(router.WithMatchedRoute(context.Background(), home)) {
		t.Fatal("expected true for default route")
	}
	if router.IsDefaultRouteContext(router.WithMatchedRoute(context.Background(), blog)) {
		t.Fatal("expected false for non-default route")
	}
}

func TestRouteFromContext(t *testing.T) {
	route := models.Route{
		RouteID:          5,
		Name:             "Items",
		Path:             "/content/item/*",
		Type:             models.RouteTypeController,
		Controller:       "content",
		ControllerAction: "item",
		ShowTitle:        false,
	}
	ctx := router.WithMatchedRoute(context.Background(), route)
	view, ok := router.RouteFromContext(ctx)
	if !ok || view.Name != "Items" || view.Controller != "content" || view.Action != "item" || view.ShowTitle {
		t.Fatalf("RouteFromContext() = %+v, ok=%v", view, ok)
	}
	if router.ShowTitleForRoute(route) {
		t.Fatal("expected ShowTitleForRoute false")
	}
}
