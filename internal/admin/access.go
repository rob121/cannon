package admin

import (
	"context"
	"net/http"
	"strings"

	"github.com/rob121/cannon/internal/models"
	"github.com/rob121/cannon/internal/roles"
	"github.com/rob121/cannon/internal/sites"
	"github.com/rob121/cannon/internal/user"
)

// AdminRoute describes one configurable admin section.
type AdminRoute struct {
	Path    string
	Label   string
	FormKey string
}

// AdminRoutes lists admin sections assignable to groups.
var AdminRoutes = []AdminRoute{
	{Path: "/items", Label: "Items", FormKey: "items"},
	{Path: "/categories", Label: "Categories", FormKey: "categories"},
	{Path: "/tags", Label: "Tags", FormKey: "tags"},
	{Path: "/field-groups", Label: "Field Groups", FormKey: "field-groups"},
	{Path: "/comments", Label: "Comments", FormKey: "comments"},
	{Path: "/media", Label: "Media", FormKey: "media"},
	{Path: "/blocks", Label: "Blocks", FormKey: "blocks"},
	{Path: "/routes", Label: "Routes", FormKey: "routes"},
	{Path: "/templates", Label: "Templates", FormKey: "templates"},
	{Path: "/menus", Label: "Menus", FormKey: "menus"},
	{Path: "/menu-items", Label: "Menu Items", FormKey: "menu-items"},
	{Path: "/users", Label: "Users", FormKey: "users"},
	{Path: "/groups", Label: "Groups", FormKey: "groups"},
	{Path: "/roles", Label: "Roles", FormKey: "roles"},
	{Path: "/notifications", Label: "Notifications", FormKey: "notifications"},
	{Path: "/configuration", Label: "Configuration", FormKey: "configuration"},
	{Path: "/extensions", Label: "Extensions", FormKey: "extensions"},
	{Path: "/help", Label: "Help", FormKey: "help"},
	{Path: "/languages", Label: "Languages", FormKey: "languages"},
	{Path: "/sites", Label: "Sites", FormKey: "sites"},
	{Path: "/system", Label: "System", FormKey: "system"},
	{Path: "/authenticators", Label: "Authenticators", FormKey: "authenticators"},
	{Path: "/profiles", Label: "Profiles", FormKey: "profiles"},
}

// CanAccessAdmin reports whether the user may access an admin path.
func CanAccessAdmin(ctx context.Context, userID uint, requestPath string, write bool) (bool, error) {
	if ok, err := roles.HasRole(ctx, userID, roles.AdminRole); err != nil {
		return false, err
	} else if ok {
		return true, nil
	}
	db, err := sites.DB(ctx)
	if err != nil {
		return false, err
	}
	var u models.User
	if err := db.Preload("Groups").First(&u, userID).Error; err != nil {
		return false, err
	}
	groupIDs := make([]uint, 0, len(u.Groups))
	for _, g := range u.Groups {
		if g.Status == models.StatusActive {
			groupIDs = append(groupIDs, g.GroupID)
		}
	}
	if len(groupIDs) == 0 {
		return false, nil
	}
	section := adminSection(requestPath)
	if section == "" {
		return false, nil
	}
	if section == "/" {
		return true, nil
	}
	var count int64
	q := db.Model(&models.GroupAdminRoute{}).
		Where("group_id IN ? AND path = ?", groupIDs, section)
	if write {
		q = q.Where("can_write = ?", true)
	} else {
		q = q.Where("can_read = ?", true)
	}
	if err := q.Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

func adminSection(path string) string {
	path = strings.TrimPrefix(path, "/admin")
	path = strings.TrimPrefix(path, "/")
	if path == "" {
		return "/"
	}
	parts := strings.Split(path, "/")
	return "/" + parts[0]
}

func (h *Handler) requireAccess(w http.ResponseWriter, r *http.Request) bool {
	svc, err := user.FromContext(r.Context())
	if err != nil {
		httpxRedirectLogin(w, r)
		return false
	}
	userID, ok := svc.CurrentID()
	if !ok {
		httpxRedirectLogin(w, r)
		return false
	}
	write := r.Method != http.MethodGet && r.Method != http.MethodHead
	allowed, err := CanAccessAdmin(r.Context(), userID, r.URL.Path, write)
	if err != nil || !allowed {
		h.forbidden(w, r)
		return false
	}
	return true
}

func httpxRedirectLogin(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "/admin/login", http.StatusSeeOther)
}
