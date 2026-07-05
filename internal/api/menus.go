package api

import (
	"net/http"
	"strings"

	"github.com/rob121/cannon/internal/models"
	"github.com/rob121/cannon/internal/router"
	"github.com/rob121/cannon/internal/sites"
)

func (h *Handler) serveMenus(w http.ResponseWriter, r *http.Request, parts []string) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}
	ctx := r.Context()
	viewerGroups, _ := resolveViewerGroups(ctx)
	if len(parts) == 0 {
		db, err := sites.DB(ctx)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "server_error", err.Error())
			return
		}
		var rows []models.Menu
		if err := db.Where("status = ?", models.StatusActive).Order("menu_name asc").Find(&rows).Error; err != nil {
			writeError(w, http.StatusInternalServerError, "server_error", err.Error())
			return
		}
		out := make([]map[string]any, 0, len(rows))
		for _, row := range rows {
			out = append(out, map[string]any{
				"menu_id": row.MenuID,
				"name":    row.MenuName,
			})
		}
		writeJSON(w, http.StatusOK, map[string]any{"data": out})
		return
	}
	name := strings.TrimSpace(parts[0])
	if name == "" {
		writeError(w, http.StatusNotFound, "not_found", "Menu not found")
		return
	}
	items, err := router.MenuDataForViewer(ctx, name, viewerGroups)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "server_error", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"name":  name,
		"items": menuItemsJSON(items),
	})
}

func menuItemsJSON(items []map[string]any) []map[string]any {
	out := make([]map[string]any, 0, len(items))
	for _, item := range items {
		out = append(out, menuItemJSON(item))
	}
	return out
}

func menuItemJSON(item map[string]any) map[string]any {
	row := map[string]any{
		"menu_item_id": item["MenuItemID"],
		"name":         item["Name"],
		"href":         item["Href"],
		"class":        item["Class"],
		"target":       item["Target"],
	}
	if children, ok := item["Children"].([]map[string]any); ok && len(children) > 0 {
		row["children"] = menuItemsJSON(children)
	}
	return row
}
