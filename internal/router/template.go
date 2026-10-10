package router

import (
	"context"
	"strings"

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

// IsDefaultRouteContext reports whether the matched route is the site default.
func IsDefaultRouteContext(ctx context.Context) bool {
	route, ok := MatchedRoute(ctx)
	return ok && route.IsDefault
}

// IsRouteContext reports whether the matched route equals controller and optional action.
func IsRouteContext(ctx context.Context, controller, action string) bool {
	route, ok := RouteFromContext(ctx)
	if !ok {
		return false
	}
	if !strings.EqualFold(strings.TrimSpace(controller), route.Controller) {
		return false
	}
	action = strings.TrimSpace(action)
	if action == "" {
		return true
	}
	return strings.EqualFold(action, route.Action)
}
