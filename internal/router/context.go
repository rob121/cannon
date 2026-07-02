package router

import (
	"context"

	"github.com/rob121/cannon/internal/models"
)

type matchedRouteKey struct{}

// WithMatchedRoute stores the resolved frontend route on the request context.
func WithMatchedRoute(ctx context.Context, route models.Route) context.Context {
	return context.WithValue(ctx, matchedRouteKey{}, route)
}

// MatchedRoute returns the resolved frontend route, if any.
func MatchedRoute(ctx context.Context) (models.Route, bool) {
	if ctx == nil {
		return models.Route{}, false
	}
	route, ok := ctx.Value(matchedRouteKey{}).(models.Route)
	if !ok || route.RouteID == 0 {
		return models.Route{}, false
	}
	return route, true
}
