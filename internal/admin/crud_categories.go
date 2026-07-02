package admin

import (
	"net/http"

	"github.com/rob121/cannon/internal/content"
	"github.com/rob121/cannon/internal/models"
	"github.com/rob121/cannon/internal/sites"
)

const categoriesBase = "/admin/categories"

func (h *Handler) categories(w http.ResponseWriter, r *http.Request, path string) {
	parts := pathParts("/categories", path)
	switch {
	case len(parts) == 0:
		h.categoryList(w, r)
	case parts[0] == "new":
		h.categoryForm(w, r, 0)
	case len(parts) == 2 && parts[1] == "delete":
		h.categoryDelete(w, r, parts[0])
	case len(parts) == 2 && parts[1] == "toggle-status":
		h.categoryToggleStatus(w, r, parts[0])
	default:
		id, ok := parseID(parts[0])
		if !ok {
			http.NotFound(w, r)
			return
		}
		h.categoryForm(w, r, id)
	}
}

func (h *Handler) categoryList(w http.ResponseWriter, r *http.Request) {
	db, _ := sites.DB(r.Context())
	page := parsePage(r)
	var total int64
	db.Model(&models.Category{}).Count(&total)
	data := listPage(page, total, categoriesBase,
		"Organize items into nested categories.",
		"Add Category", map[string]any{"ActiveNav": "categories"})
	order := applyListSort(r, data, map[string]string{
		"name": "name", "slug": "slug", "sort": "sort", "status": "status",
	}, "sort")
	var rows []models.Category
	db.Offset((page - 1) * pageSize).Limit(pageSize).Order(order).Find(&rows)
	all, _ := content.CategoryTree(r.Context())
	data["Rows"] = rows
	data["AllCategories"] = all
	h.render(w, r, "Categories", "admin/categories.html", data)
}

func (h *Handler) categoryForm(w http.ResponseWriter, r *http.Request, id uint) {
	db, _ := sites.DB(r.Context())
	isNew := id == 0
	var row models.Category
	if !isNew {
		if err := db.Preload("Groups").First(&row, id).Error; err != nil {
			http.NotFound(w, r)
			return
		}
	} else {
		row.Status = models.StatusActive
		row.InheritSettings = true
	}
	allGroups := loadActiveGroups(db)
	var fieldGroups []models.ContentFieldGroup
	db.Order("name asc").Find(&fieldGroups)
	allCats, _ := content.CategoryTree(r.Context())

	if r.Method == http.MethodPost {
		if err := r.ParseForm(); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		row.Name = formString(r, "name")
		row.Slug = formString(r, "slug")
		if row.Slug == "" {
			row.Slug, _ = content.UniqueCategorySlug(r.Context(), row.Name, row.CategoryID)
		}
		row.Description = r.FormValue("description")
		row.Image = formString(r, "image")
		row.Template = formString(r, "template")
		row.Sort = formInt(r, "sort", 0)
		row.Status = formStatus(r)
		row.InheritSettings = formBool(r, "inherit_settings")
		row.ParentID = formUintPtr(r, "parent_id")
		row.FieldGroupID = formUintPtr(r, "field_group_id")
		var saveErr error
		if isNew {
			saveErr = db.Create(&row).Error
		} else {
			saveErr = db.Save(&row).Error
		}
		if saveErr != nil {
			h.renderCategoryForm(w, r, row, allGroups, fieldGroups, allCats, isNew, saveErr.Error())
			return
		}
		if err := replaceFormGroups(db, &row, r); err != nil {
			h.renderCategoryForm(w, r, row, allGroups, fieldGroups, allCats, isNew, err.Error())
			return
		}
		redirectList(w, r, categoriesBase)
		return
	}
	h.renderCategoryForm(w, r, row, allGroups, fieldGroups, allCats, isNew, "")
}

func (h *Handler) renderCategoryForm(w http.ResponseWriter, r *http.Request, row models.Category, allGroups []models.Group, fieldGroups []models.ContentFieldGroup, allCats []models.Category, isNew bool, errMsg string) {
	title := "Add Category"
	subtitle := "Create a category for grouping items."
	if !isNew {
		title = "Edit Category"
		subtitle = "Update category details, template, and access."
	}
	data := formData(map[string]any{
		"ActiveNav":     "categories",
		"Row":           row,
		"IsNew":         isNew,
		"BasePath":      categoriesBase,
		"Subtitle":      subtitle,
		"AllGroups":     allGroups,
		"SelectedIDs":   groupSelectedIDs(row.Groups),
		"FieldGroups":   fieldGroups,
		"AllCategories": allCats,
	})
	if errMsg != "" {
		data["Error"] = errMsg
	}
	h.render(w, r, title, "admin/categories_form.html", data)
}

func (h *Handler) categoryDelete(w http.ResponseWriter, r *http.Request, idStr string) {
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
	db.Exec("DELETE FROM category_groups WHERE category_category_id = ?", id)
	if err := db.Delete(&models.Category{}, id).Error; err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	redirectList(w, r, categoriesBase)
}

func (h *Handler) categoryToggleStatus(w http.ResponseWriter, r *http.Request, idStr string) {
	h.postToggleModel(w, r, idStr, &models.Category{}, categoriesBase)
}
