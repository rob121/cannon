package admin

import (
	"net/http"

	"github.com/rob121/cannon/internal/models"
	"github.com/rob121/cannon/internal/roles"
	"github.com/rob121/cannon/internal/sites"
)

const rolesBase = "/admin/roles"

func (h *Handler) roles(w http.ResponseWriter, r *http.Request, path string) {
	parts := pathParts("/roles", path)
	switch {
	case len(parts) == 0:
		h.roleList(w, r)
	case parts[0] == "new":
		h.roleForm(w, r, 0)
	case len(parts) == 2 && parts[1] == "delete":
		h.roleDelete(w, r, parts[0])
	case len(parts) == 2 && parts[1] == "toggle-status":
		h.roleToggleStatus(w, r, parts[0])
	default:
		id, ok := parseID(parts[0])
		if !ok {
			h.notFound(w, r)
			return
		}
		h.roleForm(w, r, id)
	}
}

func (h *Handler) roleList(w http.ResponseWriter, r *http.Request) {
	db, _ := sites.DB(r.Context())
	page := parsePage(r)
	var rows []models.Role
	var total int64
	db.Model(&models.Role{}).Count(&total)
	data := listPage(r, page, total, rolesBase,
		"Permission roles assigned to user groups.",
		"Add Role", map[string]any{"ActiveNav": "roles"})
	order := applyListSort(r, data, map[string]string{
		"name": "name", "status": "status",
	}, "name")
	db.Offset((page - 1) * pageSizeFor(r)).Limit(pageSizeFor(r)).Order(order).Find(&rows)
	data["Rows"] = rows
	h.render(w, r, "Roles", "admin/roles.html", data)
}

func (h *Handler) roleForm(w http.ResponseWriter, r *http.Request, id uint) {
	db, _ := sites.DB(r.Context())
	isNew := id == 0
	var row models.Role
	if !isNew {
		if err := db.First(&row, id).Error; err != nil {
			h.notFound(w, r)
			return
		}
	}
	if r.Method == http.MethodPost {
		if err := r.ParseForm(); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		row.Name = formString(r, "name")
		row.Status = formStatus(r)
		var err error
		if isNew {
			err = db.Create(&row).Error
		} else {
			err = db.Save(&row).Error
		}
		if err != nil {
			h.render(w, r, "Role", "admin/roles_form.html", formData(map[string]any{
				"ActiveNav": "roles", "Error": err.Error(), "Row": row, "IsNew": isNew, "BasePath": rolesBase,
			}))
			return
		}
		redirectList(w, r, rolesBase)
		return
	}
	title := "Add Role"
	subtitle := "Create a permission role for group assignment."
	if !isNew {
		title = "Edit Role"
		subtitle = "Update role name and activation status."
	}
	h.render(w, r, title, "admin/roles_form.html", formData(map[string]any{
		"ActiveNav": "roles", "Row": row, "IsNew": isNew, "BasePath": rolesBase, "Subtitle": subtitle,
		"Protected": !isNew && row.Name == roles.AdminRole,
	}))
}

func (h *Handler) roleDelete(w http.ResponseWriter, r *http.Request, idStr string) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	id, ok := parseID(idStr)
	if !ok {
		h.notFound(w, r)
		return
	}
	db, _ := sites.DB(r.Context())
	var row models.Role
	if err := db.First(&row, id).Error; err != nil {
		h.notFound(w, r)
		return
	}
	if row.Name == roles.AdminRole {
		http.Error(w, "the admin role cannot be deleted", http.StatusBadRequest)
		return
	}
	if err := db.Exec("DELETE FROM group_roles WHERE role_role_id = ?", row.RoleID).Error; err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if err := db.Delete(&row).Error; err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	redirectList(w, r, rolesBase)
}
