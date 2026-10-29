package admin

import (
	"net/http"
	"strings"

	"github.com/rob121/cannon/internal/security"
)

const permissionsBase = "/admin/permissions"

func (h *Handler) permissions(w http.ResponseWriter, r *http.Request, path string) {
	if strings.TrimPrefix(path, "/permissions") != "" {
		h.notFound(w, r)
		return
	}
	all, err := permissionsForCatalog(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	filter := strings.TrimSpace(r.URL.Query().Get("category"))
	if filter != "" {
		filtered := make([]security.Permission, 0)
		for _, p := range all {
			if p.Category == filter {
				filtered = append(filtered, p)
			}
		}
		all = filtered
	}
	data := map[string]any{
		"ActiveNav":       "permissions",
		"Subtitle":        "Registered capabilities available for role assignment.",
		"Permissions":     all,
		"Categories":      security.Categories(all),
		"FilterCategory":  filter,
		"PermsByCategory": security.PermissionsByCategory(all),
	}
	h.render(w, r, "Permissions", "admin/permissions.html", data)
}
