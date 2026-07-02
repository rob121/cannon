package admin

import (
	"net/http"
	"sort"
	"strconv"

	"github.com/rob121/cannon/internal/models"
	"github.com/rob121/cannon/internal/sites"
)

const profilesBase = "/admin/profiles"

func (h *Handler) profiles(w http.ResponseWriter, r *http.Request, path string) {
	parts := pathParts("/profiles", path)
	switch {
	case len(parts) == 0:
		h.profileList(w, r)
	case parts[0] == "new":
		h.profileForm(w, r, 0)
	case len(parts) == 2 && parts[1] == "delete":
		h.profileDelete(w, r, parts[0])
	case len(parts) == 4 && parts[1] == "fields" && parts[3] == "toggle-status":
		h.profileFieldToggleStatus(w, r, parts[0], parts[2])
	case len(parts) >= 3 && parts[1] == "fields":
		h.profileFieldAction(w, r, parts)
	default:
		id, ok := parseID(parts[0])
		if !ok {
			http.NotFound(w, r)
			return
		}
		h.profileForm(w, r, id)
	}
}

func (h *Handler) profileList(w http.ResponseWriter, r *http.Request) {
	db, _ := sites.DB(r.Context())
	page := parsePage(r)
	var rows []models.Profile
	var total int64
	db.Model(&models.Profile{}).Count(&total)
	data := listPage(page, total, profilesBase,
		"User profile schemas and custom fields.",
		"Add Profile", map[string]any{"ActiveNav": "profiles"})
	order := applyListSort(r, data, map[string]string{"name": "name"}, "name")
	db.Offset((page - 1) * pageSize).Limit(pageSize).Preload("Fields").Order(order).Find(&rows)
	data["Rows"] = rows
	h.render(w, r, "Profiles", "admin/profiles.html", data)
}

func (h *Handler) profileForm(w http.ResponseWriter, r *http.Request, id uint) {
	db, _ := sites.DB(r.Context())
	isNew := id == 0
	var row models.Profile
	if !isNew {
		if err := db.Preload("Fields").First(&row, id).Error; err != nil {
			http.NotFound(w, r)
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
			h.render(w, r, "Profile", "admin/profiles_form.html", formData(map[string]any{
				"ActiveNav": "profiles", "Error": err.Error(), "Row": row, "IsNew": isNew,
			}))
			return
		}
		redirectList(w, r, profilesBase+"/"+strconv.FormatUint(uint64(row.ProfileID), 10))
		return
	}
	title := "Add Profile"
	if !isNew {
		title = "Edit Profile"
	}
	h.render(w, r, title, "admin/profiles_form.html", formData(map[string]any{
		"ActiveNav": "profiles", "Row": row, "IsNew": isNew, "BasePath": profilesBase,
	}))
}

func (h *Handler) profileFieldAction(w http.ResponseWriter, r *http.Request, parts []string) {
	profileID, ok := parseID(parts[0])
	if !ok {
		http.NotFound(w, r)
		return
	}
	db, _ := sites.DB(r.Context())
	if len(parts) == 3 && parts[2] == "new" {
		if r.Method == http.MethodPost {
			_ = r.ParseForm()
			field := models.ProfileField{
				ProfileID:     profileID,
				Name:          formString(r, "field_name"),
				Type:          formString(r, "field_type"),
				Sort:          formInt(r, "field_sort", 0),
				Status:        formStatus(r),
				Configuration: r.FormValue("field_configuration"),
			}
			if err := db.Create(&field).Error; err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		}
		redirectList(w, r, profilesBase+"/"+strconv.FormatUint(uint64(profileID), 10))
		return
	}
	if len(parts) == 4 && parts[3] == "delete" {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		fieldID, ok := parseID(parts[2])
		if !ok {
			http.NotFound(w, r)
			return
		}
		db.Where("profile_id = ?", profileID).Delete(&models.ProfileField{}, fieldID)
		redirectList(w, r, profilesBase+"/"+strconv.FormatUint(uint64(profileID), 10))
		return
	}
	if len(parts) == 3 {
		fieldID, ok := parseID(parts[2])
		if !ok {
			http.NotFound(w, r)
			return
		}
		var field models.ProfileField
		if err := db.Where("profile_id = ?", profileID).First(&field, fieldID).Error; err != nil {
			http.NotFound(w, r)
			return
		}
		if r.Method == http.MethodPost {
			_ = r.ParseForm()
			field.Name = formString(r, "name")
			field.Type = formString(r, "type")
			field.Sort = formInt(r, "sort", 0)
			field.Status = formStatus(r)
			field.Configuration = r.FormValue("configuration")
			if err := db.Save(&field).Error; err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			redirectList(w, r, profilesBase+"/"+strconv.FormatUint(uint64(profileID), 10))
			return
		}
		h.render(w, r, "Edit Profile Field", "admin/profile_field_form.html", formData(map[string]any{
			"ActiveNav": "profiles", "Field": field, "ProfileID": profileID, "BasePath": profilesBase,
		}))
		return
	}
	http.NotFound(w, r)
}

func (h *Handler) profileDelete(w http.ResponseWriter, r *http.Request, idStr string) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	id, ok := parseID(idStr)
	if !ok {
		http.NotFound(w, r)
		return
	}
	db, _ := sites.DB(r.Context())
	db.Where("profile_id = ?", id).Delete(&models.ProfileField{})
	if err := db.Delete(&models.Profile{}, id).Error; err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	redirectList(w, r, profilesBase)
}
