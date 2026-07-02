package admin

import (
	"net/http"

	"github.com/rob121/cannon/internal/content"
	"github.com/rob121/cannon/internal/models"
	"github.com/rob121/cannon/internal/sites"
)

const tagsBase = "/admin/tags"

func (h *Handler) tags(w http.ResponseWriter, r *http.Request, path string) {
	parts := pathParts("/tags", path)
	switch {
	case len(parts) == 0:
		h.tagList(w, r)
	case parts[0] == "new":
		h.tagForm(w, r, 0)
	case len(parts) == 2 && parts[1] == "delete":
		h.tagDelete(w, r, parts[0])
	default:
		id, ok := parseID(parts[0])
		if !ok {
			http.NotFound(w, r)
			return
		}
		h.tagForm(w, r, id)
	}
}

func (h *Handler) tagList(w http.ResponseWriter, r *http.Request) {
	db, _ := sites.DB(r.Context())
	page := parsePage(r)
	var total int64
	db.Model(&models.Tag{}).Count(&total)
	data := listPage(page, total, tagsBase,
		"Reusable labels for organizing and filtering items.",
		"Add Tag", map[string]any{"ActiveNav": "tags"})
	order := applyListSort(r, data, map[string]string{"name": "name", "slug": "slug"}, "name")
	var rows []models.Tag
	db.Offset((page - 1) * pageSize).Limit(pageSize).Order(order).Find(&rows)
	data["Rows"] = rows
	h.render(w, r, "Tags", "admin/tags.html", data)
}

func (h *Handler) tagForm(w http.ResponseWriter, r *http.Request, id uint) {
	db, _ := sites.DB(r.Context())
	isNew := id == 0
	var row models.Tag
	if !isNew {
		if err := db.First(&row, id).Error; err != nil {
			http.NotFound(w, r)
			return
		}
	}
	if r.Method == http.MethodPost {
		if err := r.ParseForm(); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		row.Name = formString(r, "name")
		row.Slug = formString(r, "slug")
		if row.Slug == "" {
			row.Slug, _ = content.UniqueTagSlug(r.Context(), row.Name, row.TagID)
		}
		var saveErr error
		if isNew {
			saveErr = db.Create(&row).Error
		} else {
			saveErr = db.Save(&row).Error
		}
		if saveErr != nil {
			h.renderTagForm(w, r, row, isNew, saveErr.Error())
			return
		}
		redirectList(w, r, tagsBase)
		return
	}
	h.renderTagForm(w, r, row, isNew, "")
}

func (h *Handler) renderTagForm(w http.ResponseWriter, r *http.Request, row models.Tag, isNew bool, errMsg string) {
	title := "Add Tag"
	subtitle := "Create a tag for cross-category labeling."
	if !isNew {
		title = "Edit Tag"
		subtitle = "Update tag name and slug."
	}
	data := formData(map[string]any{
		"ActiveNav": "tags",
		"Row":       row,
		"IsNew":     isNew,
		"BasePath":  tagsBase,
		"Subtitle":  subtitle,
	})
	if errMsg != "" {
		data["Error"] = errMsg
	}
	h.render(w, r, title, "admin/tags_form.html", data)
}

func (h *Handler) tagDelete(w http.ResponseWriter, r *http.Request, idStr string) {
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
	db.Exec("DELETE FROM item_tags WHERE tag_tag_id = ?", id)
	if err := db.Delete(&models.Tag{}, id).Error; err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	redirectList(w, r, tagsBase)
}
