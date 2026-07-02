package admin

import (
	"net/http"
	"sort"
	"strconv"

	"github.com/rob121/cannon/internal/models"
	"github.com/rob121/cannon/internal/sites"
)

const fieldGroupsBase = "/admin/field-groups"

func (h *Handler) fieldGroups(w http.ResponseWriter, r *http.Request, path string) {
	parts := pathParts("/field-groups", path)
	switch {
	case len(parts) == 0:
		h.fieldGroupList(w, r)
	case parts[0] == "new":
		h.fieldGroupForm(w, r, 0)
	case len(parts) == 2 && parts[1] == "delete":
		h.fieldGroupDelete(w, r, parts[0])
	case len(parts) >= 3 && parts[1] == "fields":
		h.contentFieldAction(w, r, parts)
	default:
		id, ok := parseID(parts[0])
		if !ok {
			h.notFound(w, r)
			return
		}
		h.fieldGroupForm(w, r, id)
	}
}

func (h *Handler) fieldGroupList(w http.ResponseWriter, r *http.Request) {
	db, _ := sites.DB(r.Context())
	page := parsePage(r)
	var total int64
	db.Model(&models.ContentFieldGroup{}).Count(&total)
	data := listPage(r, page, total, fieldGroupsBase,
		"Custom field groups assignable to categories.",
		"Add Field Group", map[string]any{"ActiveNav": "field_groups"})
	order := applyListSort(r, data, map[string]string{"name": "name"}, "name")
	var rows []models.ContentFieldGroup
	db.Offset((page - 1) * pageSizeFor(r)).Limit(pageSizeFor(r)).Preload("Fields").Order(order).Find(&rows)
	data["Rows"] = rows
	h.render(w, r, "Field Groups", "admin/field_groups.html", data)
}

func (h *Handler) fieldGroupForm(w http.ResponseWriter, r *http.Request, id uint) {
	db, _ := sites.DB(r.Context())
	isNew := id == 0
	var row models.ContentFieldGroup
	if !isNew {
		if err := db.Preload("Fields").First(&row, id).Error; err != nil {
			h.notFound(w, r)
			return
		}
		sort.Slice(row.Fields, func(i, j int) bool { return row.Fields[i].Sort < row.Fields[j].Sort })
	}
	if r.Method == http.MethodPost {
		if err := r.ParseForm(); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		row.Name = formString(r, "name")
		var err error
		if isNew {
			err = db.Create(&row).Error
		} else {
			err = db.Save(&row).Error
		}
		if err != nil {
			h.render(w, r, "Field Group", "admin/field_groups_form.html", formData(map[string]any{
				"ActiveNav": "field_groups", "Error": err.Error(), "Row": row, "IsNew": isNew, "BasePath": fieldGroupsBase,
			}))
			return
		}
		redirectList(w, r, fieldGroupsBase+"/"+strconv.FormatUint(uint64(row.FieldGroupID), 10))
		return
	}
	title := "Add Field Group"
	if !isNew {
		title = "Edit Field Group"
	}
	h.render(w, r, title, "admin/field_groups_form.html", formData(map[string]any{
		"ActiveNav": "field_groups", "Row": row, "IsNew": isNew, "BasePath": fieldGroupsBase,
	}))
}

func (h *Handler) contentFieldAction(w http.ResponseWriter, r *http.Request, parts []string) {
	groupID, ok := parseID(parts[0])
	if !ok {
		h.notFound(w, r)
		return
	}
	db, _ := sites.DB(r.Context())
	if len(parts) == 3 && parts[2] == "new" {
		h.contentFieldForm(w, r, groupID, 0)
		return
	}
	fieldID, ok := parseID(parts[2])
	if !ok {
		h.notFound(w, r)
		return
	}
	if len(parts) == 4 && parts[3] == "delete" {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		db.Delete(&models.ContentField{}, fieldID)
		redirectList(w, r, fieldGroupsBase+"/"+strconv.FormatUint(uint64(groupID), 10))
		return
	}
	if len(parts) == 4 && parts[3] == "toggle-status" {
		h.postToggleModel(w, r, parts[2], &models.ContentField{}, fieldGroupsBase+"/"+strconv.FormatUint(uint64(groupID), 10))
		return
	}
	h.contentFieldForm(w, r, groupID, fieldID)
}

func (h *Handler) contentFieldForm(w http.ResponseWriter, r *http.Request, groupID, fieldID uint) {
	db, _ := sites.DB(r.Context())
	isNew := fieldID == 0
	var row models.ContentField
	if !isNew {
		if err := db.First(&row, fieldID).Error; err != nil {
			h.notFound(w, r)
			return
		}
	} else {
		row.FieldGroupID = groupID
		row.Status = models.StatusActive
		row.Type = "text"
	}
	if r.Method == http.MethodPost {
		if err := r.ParseForm(); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		row.FieldGroupID = groupID
		if postedGroup, ok := parseID(r.FormValue("field_group_id")); ok && postedGroup > 0 {
			row.FieldGroupID = postedGroup
		}
		if row.FieldGroupID != groupID {
			h.render(w, r, "Field", "admin/content_field_form.html", formData(map[string]any{
				"ActiveNav": "field_groups", "Error": "field group mismatch", "Row": row, "IsNew": isNew,
				"BasePath": fieldGroupsBase, "GroupID": groupID,
			}))
			return
		}
		row.Name = formString(r, "name")
		row.Label = formString(r, "label")
		row.Type = formString(r, "type")
		row.Sort = formInt(r, "sort", 0)
		row.Required = formBool(r, "required")
		row.Status = formStatus(r)
		row.Configuration = r.FormValue("configuration")
		var err error
		if isNew {
			err = db.Create(&row).Error
		} else {
			err = db.Save(&row).Error
		}
		if err != nil {
			h.render(w, r, "Field", "admin/content_field_form.html", formData(map[string]any{
				"ActiveNav": "field_groups", "Error": err.Error(), "Row": row, "IsNew": isNew,
				"BasePath": fieldGroupsBase, "GroupID": groupID,
			}))
			return
		}
		redirectList(w, r, fieldGroupsBase+"/"+strconv.FormatUint(uint64(groupID), 10))
		return
	}
	title := "Add Field"
	if !isNew {
		title = "Edit Field"
	}
	h.render(w, r, title, "admin/content_field_form.html", formData(map[string]any{
		"ActiveNav": "field_groups", "Row": row, "IsNew": isNew, "BasePath": fieldGroupsBase, "GroupID": groupID,
	}))
}

func (h *Handler) fieldGroupDelete(w http.ResponseWriter, r *http.Request, idStr string) {
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
	db.Where("field_group_id = ?", id).Delete(&models.ContentField{})
	if err := db.Delete(&models.ContentFieldGroup{}, id).Error; err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	redirectList(w, r, fieldGroupsBase)
}
