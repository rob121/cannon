package sites

import (
	"context"

	"github.com/rob121/cannon/internal/models"
)

const fallbackDefaultRoutePath = "/"

// DefaultRoutePath returns the path of the active default site route, or "/" when unset.
func DefaultRoutePath(ctx context.Context) string {
	db, err := DB(ctx)
	if err != nil {
		return fallbackDefaultRoutePath
	}
	var route models.Route
	err = db.Where("is_default = ? AND status = ?", true, models.StatusActive).First(&route).Error
	if err != nil || route.Path == "" {
		return fallbackDefaultRoutePath
	}
	return route.Path
}
