package router_test

import (
	"context"
	"testing"

	"github.com/rob121/cannon/internal/models"
	"github.com/rob121/cannon/internal/router"
)

func TestMatchedRouteContext(t *testing.T) {
	route := models.Route{RouteID: 42, Name: "Home", Path: "/"}
	ctx := router.WithMatchedRoute(context.Background(), route)
	got, ok := router.MatchedRoute(ctx)
	if !ok || got.RouteID != 42 {
		t.Fatalf("MatchedRoute() = %+v, ok=%v", got, ok)
	}
}
