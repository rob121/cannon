package admin

import (
	"net/http"

	"github.com/rob121/cannon/internal/models"
	"github.com/rob121/cannon/internal/sites"
)

func (h *Handler) system(w http.ResponseWriter, r *http.Request, path string) {
	parts := pathParts("/system", path)
	switch {
	case len(parts) == 1 && parts[0] == "reload":
		h.systemReload(w, r)
	default:
		h.notFound(w, r)
	}
}

func (h *Handler) systemReload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if h.reload == nil {
		http.Error(w, "reload unavailable", http.StatusInternalServerError)
		return
	}
	data := map[string]any{
		"ActiveNav": "dashboard",
		"Subtitle":  dashboardWelcomeSubtitle(r),
	}
	if err := h.reload(); err != nil {
		data["Error"] = "Reload failed: " + err.Error()
	} else {
		data["Success"] = "Application reloaded. Routes, configuration, and extensions were refreshed without restarting the process."
	}
	h.renderDashboard(w, r, data)
}

func (h *Handler) renderDashboard(w http.ResponseWriter, r *http.Request, data map[string]any) {
	db, _ := sites.DB(r.Context())
	var userCount, routeCount, activeRouteCount, extCount int64
	var itemCount, categoryCount, tagCount, commentCount, trashedCount int64
	db.Model(&models.User{}).Count(&userCount)
	db.Model(&models.Route{}).Count(&routeCount)
	db.Model(&models.Route{}).Where("status = ?", models.StatusActive).Count(&activeRouteCount)
	db.Model(&models.Extension{}).Count(&extCount)
	db.Model(&models.Item{}).Where("status <> ?", models.ItemStatusTrashed).Count(&itemCount)
	db.Model(&models.Category{}).Count(&categoryCount)
	db.Model(&models.Tag{}).Count(&tagCount)
	db.Model(&models.Comment{}).Where("approved = ?", false).Count(&commentCount)
	db.Model(&models.Item{}).Where("status = ?", models.ItemStatusTrashed).Count(&trashedCount)
	cfg := h.chain.Sites.Config()
	data["UserCount"] = userCount
	data["RouteCount"] = routeCount
	data["ActiveRouteCount"] = activeRouteCount
	data["ExtensionCount"] = extCount
	data["ItemCount"] = itemCount
	data["CategoryCount"] = categoryCount
	data["TagCount"] = tagCount
	data["PendingCommentCount"] = commentCount
	data["TrashedItemCount"] = trashedCount
	data["SiteCount"] = len(cfg.Sites)
	if _, ok := data["Subtitle"]; !ok {
		data["Subtitle"] = dashboardWelcomeSubtitle(r)
	}
	h.render(w, r, "Dashboard", "admin/dashboard.html", data)
}
