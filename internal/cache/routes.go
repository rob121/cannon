package cache

import (
	"context"
	"sync"

	"github.com/rob121/cannon/internal/models"
	"github.com/rob121/cannon/internal/sites"
	"gorm.io/gorm"
)

type cachedRoute struct {
	Route models.Route
}

var (
	routesMu     sync.RWMutex
	routesBySite = map[string][]cachedRoute{}
)

// ActiveRoutes returns active routes with groups, using the site cache when enabled.
func ActiveRoutes(ctx context.Context, db *gorm.DB) ([]models.Route, error) {
	siteID := SiteIDFromContext(ctx)
	if siteID == "" || !Enabled(ctx) {
		return loadActiveRoutes(db)
	}

	routesMu.RLock()
	cached, ok := routesBySite[siteID]
	routesMu.RUnlock()
	if ok {
		return cloneRoutes(cached), nil
	}

	routesMu.Lock()
	defer routesMu.Unlock()
	if cached, ok = routesBySite[siteID]; ok {
		return cloneRoutes(cached), nil
	}

	routes, err := loadActiveRoutes(db)
	if err != nil {
		return nil, err
	}
	stored := make([]cachedRoute, len(routes))
	for i, route := range routes {
		stored[i] = cachedRoute{Route: route}
	}
	routesBySite[siteID] = stored
	return routes, nil
}

func loadActiveRoutes(db *gorm.DB) ([]models.Route, error) {
	var routes []models.Route
	if err := db.Preload("Groups").Where("status = ?", models.StatusActive).Find(&routes).Error; err != nil {
		return nil, err
	}
	return cloneRoutesFromModels(routes), nil
}

// InvalidateRoutes drops cached routes for a site.
func InvalidateRoutes(siteID string) {
	if siteID == "" {
		return
	}
	routesMu.Lock()
	delete(routesBySite, siteID)
	routesMu.Unlock()
}

func invalidateRoutes(siteID string) {
	InvalidateRoutes(siteID)
}

func cloneRoutes(cached []cachedRoute) []models.Route {
	out := make([]models.Route, len(cached))
	for i, row := range cached {
		out[i] = cloneRoute(row.Route)
	}
	return out
}

func cloneRoutesFromModels(routes []models.Route) []models.Route {
	out := make([]models.Route, len(routes))
	for i, route := range routes {
		out[i] = cloneRoute(route)
	}
	return out
}

func cloneRoute(route models.Route) models.Route {
	out := route
	if len(route.Groups) > 0 {
		out.Groups = make([]models.Group, len(route.Groups))
		copy(out.Groups, route.Groups)
	}
	return out
}

// WarmRoutes preloads the route cache for a site context.
func WarmRoutes(ctx context.Context) error {
	db, err := sites.DB(ctx)
	if err != nil {
		return err
	}
	_, err = ActiveRoutes(ctx, db)
	return err
}
