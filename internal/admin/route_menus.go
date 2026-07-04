package admin

import (
	"fmt"
	"strings"

	"github.com/rob121/cannon/internal/models"
	"github.com/rob121/cannon/internal/router"
	"gorm.io/gorm"
)

type routeMenuLink struct {
	MenuItemID uint
	MenuID     uint
	MenuName   string
	ItemName   string
}

func loadRouteMenuLinks(db *gorm.DB, routeID uint) []routeMenuLink {
	if routeID == 0 {
		return nil
	}
	var items []models.MenuItem
	if err := db.Where("route_id = ?", routeID).Order("menu_id asc, sort asc, menu_item_id asc").Find(&items).Error; err != nil {
		return nil
	}
	if len(items) == 0 {
		return nil
	}
	menuNames := menuNamesByID(db, items)
	out := make([]routeMenuLink, 0, len(items))
	for _, item := range items {
		out = append(out, routeMenuLink{
			MenuItemID: item.MenuItemID,
			MenuID:     item.MenuID,
			MenuName:   menuNames[item.MenuID],
			ItemName:   item.Name,
		})
	}
	return out
}

func addRouteToMenu(db *gorm.DB, route models.Route, menuID uint, parentID *uint, name string) (bool, error) {
	if menuID == 0 || route.RouteID == 0 {
		return false, nil
	}
	var existing int64
	if err := db.Model(&models.MenuItem{}).Where("menu_id = ? AND route_id = ?", menuID, route.RouteID).Count(&existing).Error; err != nil {
		return false, err
	}
	if existing > 0 {
		return false, nil
	}

	name = strings.TrimSpace(name)
	if name == "" {
		name = route.Name
	}

	menuItems, err := loadMenuItemsForMenu(db, menuID)
	if err != nil {
		return false, err
	}
	item := models.MenuItem{
		MenuID:   menuID,
		ParentID: parentID,
		Name:     name,
		RouteID:  &route.RouteID,
		Sort:     nextMenuItemSort(db, menuID, parentID),
	}
	if err := router.ValidateMenuItemParent(menuItems, item, parentID); err != nil {
		return false, fmt.Errorf("choose a valid parent item in the selected menu")
	}
	if err := db.Create(&item).Error; err != nil {
		return false, err
	}
	return true, nil
}

func nextMenuItemSort(db *gorm.DB, menuID uint, parentID *uint) int {
	q := db.Model(&models.MenuItem{}).Where("menu_id = ?", menuID)
	if parentID == nil || *parentID == 0 {
		q = q.Where("parent_id IS NULL OR parent_id = 0")
	} else {
		q = q.Where("parent_id = ?", *parentID)
	}
	var maxSort int
	q.Select("COALESCE(MAX(sort), -1)").Scan(&maxSort)
	return maxSort + 1
}
