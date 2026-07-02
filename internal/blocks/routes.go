package blocks

import (
	"context"
	"slices"
	"strings"

	"github.com/rob121/cannon/internal/router"
)

const (
	RouteModeAll    = "all"
	RouteModeNone   = "none"
	RouteModeOnly   = "only"
	RouteModeExcept = "except"
)

// RouteVisible reports whether a block should render for the current request route.
func RouteVisible(meta Metadata, routeID uint) bool {
	mode := strings.TrimSpace(meta.RouteMode)
	if mode == "" || mode == RouteModeAll {
		return true
	}
	if mode == RouteModeNone {
		return false
	}
	inList := slices.Contains(meta.RouteIDs, routeID)
	switch mode {
	case RouteModeOnly:
		return routeID > 0 && inList
	case RouteModeExcept:
		return routeID == 0 || !inList
	default:
		return true
	}
}

func currentRouteID(ctx context.Context) uint {
	route, ok := router.MatchedRoute(ctx)
	if !ok {
		return 0
	}
	return route.RouteID
}
