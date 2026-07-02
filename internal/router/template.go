package router

import (
	"context"

	"github.com/rob121/cannon/internal/models"
)

// RouteView is template-safe matched route metadata for frontend layouts.
type RouteView struct {
	ID         uint
	Name       string
	Path       string
	Type       models.RouteType
	Controller string
	Action     string
	IsDefault  bool
	ShowTitle  bool
}

// RouteFromContext returns matched route metadata for templates.
func RouteFromContext(ctx context.Context) (RouteView, bool) {
	route, ok := MatchedRoute(ctx)
	if !ok {
		return RouteView{}, false
	}
	return RouteView{
		ID:         route.RouteID,
		Name:       route.Name,
		Path:       route.Path,
		Type:       route.Type,
		Controller: route.Controller,
		Action:     route.ControllerAction,
		IsDefault:  route.IsDefault,
		ShowTitle:  route.ShowTitle,
	}, true
}

// ShowTitleForRoute reports whether a matched route should render its page title.
func ShowTitleForRoute(route models.Route) bool {
	if route.RouteID == 0 {
		return true
	}
	return route.ShowTitle
}
func IsDefaultRouteContext(ctx context.Context) bool {
	route, ok := MatchedRoute(ctx)
	return ok && route.IsDefault
}
