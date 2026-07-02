package router

import (
	"github.com/rob121/cannon/internal/models"
)

func routeHookArgs(route models.Route) map[string]any {
	return map[string]any{
		"route_id":   route.RouteID,
		"route_name": route.Name,
		"route_type": route.Type,
		"route_path": route.Path,
	}
}
