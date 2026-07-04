package admin

import (
	"net/http"

	"github.com/rob121/cannon/internal/models"
	"github.com/rob121/cannon/internal/security"
	"github.com/rob121/cannon/internal/sites"
)

const groupsBase = "/admin/groups"

func (h *Handler) groups(w http.ResponseWriter, r *http.Request, path string) {
	parts := pathParts("/groups", path)
	switch {
	case len(parts) == 0:
		h.groupList(w, r)
	case parts[0] == "new":
		h.groupForm(w, r, 0)
	case len(parts) == 2 && parts[1] == "delete":
		h.groupDelete(w, r, parts[0])
	case len(parts) == 2 && parts[1] == "toggle-status":
		h.groupToggleStatus(w, r, parts[0])
	default:
		id, ok := parseID(parts[0])
		if !ok {
			h.notFound(w, r)
			return
		}
		h.groupForm(w, r, id)
	}
}

func (h *Handler) groupList(w http.ResponseWriter, r *http.Request) {
	db, _ := sites.DB(r.Context())
	page := parsePage(r)
	var rows []models.Group
	var total int64
	db.Model(&models.Group{}).Count(&total)
	data := listPage(r, page, total, groupsBase,
		"Organizational groups that assign roles to members.",
		"Add Group", map[string]any{"ActiveNav": "groups"})
	order := applyListSort(r, data, map[string]string{
		"name": "name", "status": "status",
	}, "name")
	db.Offset((page - 1) * pageSizeFor(r)).Limit(pageSizeFor(r)).Preload("Roles").Preload("Parent").Order(order).Find(&rows)
	data["Rows"] = rows
	h.render(w, r, "Groups", "admin/groups.html", data)
}

func (h *Handler) groupForm(w http.ResponseWriter, r *http.Request, id uint) {
	db, _ := sites.DB(r.Context())
	isNew := id == 0
	var row models.Group
	if !isNew {
		if err := db.Preload("Roles").First(&row, id).Error; err != nil {
			h.notFound(w, r)
			return
		}
	}
	var allRoles []models.Role
	db.Where("status = ?", models.StatusActive).Order("name asc").Find(&allRoles)
	var allGroups []models.Group
	allGroups = loadMembershipGroups(db)

	if r.Method == http.MethodPost {
		if err := r.ParseForm(); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		row.Name = formString(r, "name")
		row.Description = formString(r, "description")
		row.Status = formStatus(r)
		row.Kind = models.GroupKind(formString(r, "kind"))
		if row.Kind == "" {
			row.Kind = models.GroupKindBackend
		}
		row.ParentID = formUintPtr(r, "parent_id")
		var err error
		if isNew {
			err = db.Create(&row).Error
		} else {
			err = db.Save(&row).Error
		}
		if err != nil {
			h.renderGroupForm(w, r, row, allRoles, allGroups, isNew, err.Error())
			return
		}
		roleIDs := formUintList(r, "role_ids")
		var selected []models.Role
		if len(roleIDs) > 0 {
			db.Where("role_id IN ?", roleIDs).Find(&selected)
		}
		if err := db.Model(&row).Association("Roles").Replace(selected); err != nil {
			h.renderGroupForm(w, r, row, allRoles, allGroups, isNew, err.Error())
			return
		}
		security.InvalidateSiteContext(r.Context())
		redirectList(w, r, groupsBase)
		return
	}
	h.renderGroupForm(w, r, row, allRoles, allGroups, isNew, "")
}

func (h *Handler) renderGroupForm(w http.ResponseWriter, r *http.Request, row models.Group, allRoles []models.Role, allGroups []models.Group, isNew bool, errMsg string) {
	selected := make([]uint, 0, len(row.Roles))
	for _, role := range row.Roles {
		selected = append(selected, role.RoleID)
	}
	title := "Add Group"
	subtitle := "Create an organizational group and assign roles."
	if !isNew {
		title = "Edit Group"
		subtitle = "Update group details and role assignments."
	}
	data := formData(map[string]any{
		"ActiveNav":   "groups",
		"Row":         row,
		"IsNew":       isNew,
		"BasePath":    groupsBase,
		"Subtitle":    subtitle,
		"AllRoles":    allRoles,
		"AllGroups":   allGroups,
		"SelectedIDs": selected,
		"Protected":   !isNew && isProtectedGroupName(row.Name),
	})
	if errMsg != "" {
		data["Error"] = errMsg
	}
	h.render(w, r, title, "admin/groups_form.html", data)
}

func (h *Handler) groupDelete(w http.ResponseWriter, r *http.Request, idStr string) {
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
	var row models.Group
	if err := db.First(&row, id).Error; err != nil {
		h.notFound(w, r)
		return
	}
	if isProtectedGroupName(row.Name) {
		http.Error(w, "the "+row.Name+" group cannot be deleted", http.StatusBadRequest)
		return
	}
	if err := db.Exec("DELETE FROM group_roles WHERE group_group_id = ?", row.GroupID).Error; err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if err := db.Exec("DELETE FROM user_groups WHERE group_group_id = ?", row.GroupID).Error; err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if err := db.Delete(&row).Error; err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	security.InvalidateSiteContext(r.Context())
	redirectList(w, r, groupsBase)
}
