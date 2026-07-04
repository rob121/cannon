package admin

import (
	"context"
	"net/http"
	"net/url"
	"strings"

	"github.com/rob121/cannon/internal/security"
	"github.com/rob121/cannon/internal/user"
)

// AdminRoute describes one configurable admin section.
type AdminRoute struct {
	Path    string
	Label   string
	FormKey string
}

// AdminRoutes lists admin sections used for permission path mapping.
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
	{Path: "/permissions", Label: "Permissions", FormKey: "permissions"},
	{Path: "/notifications", Label: "Notifications", FormKey: "notifications"},
	{Path: "/configuration", Label: "Configuration", FormKey: "configuration"},
	{Path: "/extensions", Label: "Extensions", FormKey: "extensions"},
	{Path: "/extension-apps", Label: "Extension Apps", FormKey: "extension-apps"},
	{Path: "/help", Label: "Help", FormKey: "help"},
	{Path: "/languages", Label: "Languages", FormKey: "languages"},
	{Path: "/sites", Label: "Sites", FormKey: "sites"},
	{Path: "/system", Label: "System", FormKey: "system"},
	{Path: "/authenticators", Label: "Authenticators", FormKey: "authenticators"},
	{Path: "/profiles", Label: "Profiles", FormKey: "profiles"},
	{Path: "/api", Label: "API", FormKey: "api"},
}

// CanAccessAdmin reports whether the user may access an admin path.
func CanAccessAdmin(ctx context.Context, userID uint, requestPath string, write bool) (bool, error) {
	if name := extensionNameFromAdminPath(requestPath); name != "" {
		return security.CanAccessExtensionAdmin(ctx, userID, name)
	}
	section := adminSection(requestPath)
	if section == "/extension-apps" {
		return security.Can(ctx, userID, security.PermAdminExtensionAppsRead)
	}
	perm := security.AdminPermissionForPath(section, write)
	return security.Can(ctx, userID, perm)
}

func extensionNameFromAdminPath(requestPath string) string {
	path := strings.TrimPrefix(requestPath, "/admin")
	path = strings.TrimPrefix(path, "/")
	parts := strings.Split(path, "/")
	if len(parts) < 2 || parts[0] != "extension-apps" {
		return ""
	}
	name, err := url.PathUnescape(parts[1])
	if err != nil {
		return ""
	}
	return strings.TrimSpace(name)
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

// CanManageSecurity reports whether the user may manage roles and permissions.
func CanManageSecurity(ctx context.Context, userID uint) (bool, error) {
	ok, err := security.Can(ctx, userID, security.PermRolesManage)
	if err != nil || ok {
		return ok, err
	}
	return security.Can(ctx, userID, security.PermWildcardAll)
}
