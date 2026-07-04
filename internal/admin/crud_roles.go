package admin

import (
	"net/http"

	"github.com/rob121/cannon/internal/models"
	"github.com/rob121/cannon/internal/notifications"
	"github.com/rob121/cannon/internal/security"
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
		"Capability roles assigned to groups and users.",
		"Add Role", map[string]any{"ActiveNav": "roles"})
	order := applyListSort(r, data, map[string]string{
		"name": "name", "status": "status",
	}, "name")
	db.Offset((page - 1) * pageSizeFor(r)).Limit(pageSizeFor(r)).Order(order).Find(&rows)

	type roleRow struct {
		models.Role
		PermissionCount int
	}
	list := make([]roleRow, 0, len(rows))
	for _, row := range rows {
		var count int64
		db.Model(&models.RolePermission{}).Where("role_id = ?", row.RoleID).Count(&count)
		list = append(list, roleRow{Role: row, PermissionCount: int(count)})
	}
	data["Rows"] = list
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
	allRoles := []models.Role{}
	db.Where("status = ?", models.StatusActive).Order("name asc").Find(&allRoles)
	allPerms := security.RegisteredPermissions()
	categories := security.Categories(allPerms)
	permsByCategory := security.PermissionsByCategory(allPerms)
	selectedPerms, selectedDenied, _ := security.LoadRolePermissionAssignments(db, row.RoleID)
	parentIDs, _ := security.LoadParentRoleIDs(db, row.RoleID)
	selectedRoleEvents, _ := notifications.RoleSubscriptionEvents(db, row.RoleID)

	if r.Method == http.MethodPost {
		if err := r.ParseForm(); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		var err error
		if !isNew && row.SystemRole {
			priorName := row.Name
			priorStatus := row.Status
			row.Description = formString(r, "description")
			row.Name = priorName
			row.Status = priorStatus
			err = db.Save(&row).Error
		} else {
			row.Name = formString(r, "name")
			row.Description = formString(r, "description")
			row.Status = formStatus(r)
			if isNew {
				err = db.Create(&row).Error
			} else {
				err = db.Save(&row).Error
			}
		}
		if err != nil {
			h.renderRoleForm(w, r, row, allRoles, categories, permsByCategory, selectedPerms, selectedDenied, parentIDs, selectedRoleEvents, isNew, err.Error())
			return
		}
		if err := security.SaveRolePermissions(db, row.RoleID, r.Form["permission_keys"], r.Form["deny_permission_keys"]); err != nil {
			h.renderRoleForm(w, r, row, allRoles, categories, permsByCategory, selectedPerms, selectedDenied, parentIDs, selectedRoleEvents, isNew, err.Error())
			return
		}
		if err := security.SaveRoleInheritance(db, row.RoleID, formUintList(r, "parent_role_ids")); err != nil {
			h.renderRoleForm(w, r, row, allRoles, categories, permsByCategory, selectedPerms, selectedDenied, parentIDs, selectedRoleEvents, isNew, err.Error())
			return
		}
		if err := notifications.ReplaceRoleSubscriptions(db, row.RoleID, r.Form["role_notification_events"]); err != nil {
			h.renderRoleForm(w, r, row, allRoles, categories, permsByCategory, selectedPerms, selectedDenied, parentIDs, selectedRoleEvents, isNew, err.Error())
			return
		}
		security.InvalidateSiteContext(r.Context())
		redirectList(w, r, rolesBase)
		return
	}
	title := "Add Role"
	subtitle := "Create a role and assign permissions."
	if !isNew {
		title = "Edit Role"
		subtitle = "Update role capabilities, inheritance, and direct permission assignments."
	}
	h.renderRoleForm(w, r, row, allRoles, categories, permsByCategory, selectedPerms, selectedDenied, parentIDs, selectedRoleEvents, isNew, "")
	_ = title
	_ = subtitle
}

func (h *Handler) renderRoleForm(w http.ResponseWriter, r *http.Request, row models.Role, allRoles []models.Role, categories []string, permsByCategory map[string][]security.Permission, selectedPerms, selectedDenied []string, parentIDs []uint, selectedRoleEvents []string, isNew bool, errMsg string) {
	title := "Add Role"
	subtitle := "Create a role and assign permissions."
	if !isNew {
		title = "Edit Role"
		subtitle = "Update role capabilities, inheritance, and direct permission assignments."
	}
	data := formData(map[string]any{
		"ActiveNav":         "roles",
		"Row":               row,
		"IsNew":             isNew,
		"BasePath":          rolesBase,
		"Subtitle":          subtitle,
		"AllRoles":          allRoles,
		"Categories":        categories,
		"PermsByCategory":   permsByCategory,
		"SelectedPermKeys":       selectedPerms,
		"SelectedDeniedPermKeys": selectedDenied,
		"SelectedParentIDs": parentIDs,
		"SelectedRoleNotificationEvents": selectedRoleEvents,
		"SubscribableGroups":             notifications.EventGroups(),
		"Protected":         !isNew && row.SystemRole,
	})
	if errMsg != "" {
		data["Error"] = errMsg
	}
	h.render(w, r, title, "admin/roles_form.html", data)
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
	if row.SystemRole {
		http.Error(w, "system roles cannot be deleted", http.StatusBadRequest)
		return
	}
	if err := db.Exec("DELETE FROM group_roles WHERE role_role_id = ?", row.RoleID).Error; err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if err := db.Exec("DELETE FROM user_roles WHERE role_role_id = ?", row.RoleID).Error; err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if err := db.Where("role_id = ?", row.RoleID).Delete(&models.RolePermission{}).Error; err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if err := db.Where("child_role_id = ? OR parent_role_id = ?", row.RoleID, row.RoleID).
		Delete(&models.RoleInheritance{}).Error; err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	_ = notifications.DeleteRoleSubscriptions(db, row.RoleID)
	if err := db.Delete(&row).Error; err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	security.InvalidateSiteContext(r.Context())
	redirectList(w, r, rolesBase)
}
