package router

import (
	"github.com/rob121/cannon/internal/models"
	"gorm.io/gorm"
)

// SetRouteDefault marks one route as the site default and clears the flag on all others.
func SetRouteDefault(db *gorm.DB, routeID uint) error {
	return db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&models.Route{}).Where("route_id <> ?", routeID).Update("is_default", false).Error; err != nil {
			return err
		}
		return tx.Model(&models.Route{}).Where("route_id = ?", routeID).Update("is_default", true).Error
	})
}

// EnsureRouteDefault clears duplicate default flags. It does not auto-assign a default route.
func EnsureRouteDefault(db *gorm.DB) error {
	var count int64
	if err := db.Model(&models.Route{}).Where("is_default = ?", true).Count(&count).Error; err != nil {
		return err
	}
	if count == 1 {
		return nil
	}
	if count > 1 {
		var routes []models.Route
		if err := db.Where("is_default = ?", true).Order("route_id asc").Find(&routes).Error; err != nil {
			return err
		}
		for i := 1; i < len(routes); i++ {
			if err := db.Model(&routes[i]).Update("is_default", false).Error; err != nil {
				return err
			}
		}
		return nil
	}

	return nil
}
