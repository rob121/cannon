package admin

import (
	"net/http"
	"sort"
	"strings"

	"github.com/rob121/cannon/internal/models"
	"github.com/rob121/cannon/internal/security"
	"github.com/rob121/cannon/internal/sites"
)

const permissionsBase = "/admin/permissions"

func (h *Handler) permissions(w http.ResponseWriter, r *http.Request, path string) {
	if strings.TrimPrefix(path, "/permissions") != "" {
		h.notFound(w, r)
		return
	}
	db, _ := sites.DB(r.Context())
	var dbPerms []models.Permission
	db.Order("category asc, key asc").Find(&dbPerms)
	registered := security.RegisteredPermissions()
	byKey := map[string]security.Permission{}
	for _, p := range registered {
		byKey[p.ID] = p
	}
	for _, row := range dbPerms {
		if _, ok := byKey[row.Key]; !ok {
			byKey[row.Key] = security.Permission{
				ID:          row.Key,
				DisplayName: row.DisplayName,
				Description: row.Description,
				Category:    row.Category,
				Dangerous:   row.Dangerous,
			}
		}
	}
	all := make([]security.Permission, 0, len(byKey))
	for _, p := range byKey {
		all = append(all, p)
	}
	sort.Slice(all, func(i, j int) bool {
		if all[i].Category == all[j].Category {
			return all[i].ID < all[j].ID
		}
		return all[i].Category < all[j].Category
	})
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
