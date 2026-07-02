package admin

import (
	"context"
	"net/http"
	"strings"

	"github.com/rob121/cannon/internal/models"
	"github.com/rob121/cannon/internal/roles"
	"gorm.io/gorm"
)

func loadGroupAdminRoutes(db *gorm.DB, groupID uint) map[string]models.GroupAdminRoute {
	out := map[string]models.GroupAdminRoute{}
	if groupID == 0 {
		return out
	}
	var rows []models.GroupAdminRoute
	db.Where("group_id = ?", groupID).Find(&rows)
	for _, row := range rows {
		out[row.Path] = row
	}
	return out
}

func saveGroupAdminRoutes(db *gorm.DB, groupID uint, r *http.Request) error {
	if err := db.Where("group_id = ?", groupID).Delete(&models.GroupAdminRoute{}).Error; err != nil {
		return err
	}
	for _, route := range AdminRoutes {
		key := route.FormKey
		if key == "" {
			key = adminRouteFormKey(route.Path)
		}
		read := r.FormValue("route_read_"+key) != ""
		write := r.FormValue("route_write_"+key) != ""
		if !read && !write {
			continue
		}
		row := models.GroupAdminRoute{
			GroupID:  groupID,
			Path:     route.Path,
			CanRead:  read,
			CanWrite: write || read,
		}
		if err := db.Create(&row).Error; err != nil {
			return err
		}
	}
	return nil
}

func adminRouteFormKey(path string) string {
	return strings.TrimPrefix(strings.ReplaceAll(path, "/", "_"), "_")
}

func canManageGroupPermissions(ctx context.Context, userID uint) bool {
	ok, err := roles.HasRole(ctx, userID, roles.AdminRole)
	return err == nil && ok
}
