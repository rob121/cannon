package admin

import (
	"net/http"
	"sort"
	"strings"

	"github.com/rob121/cannon/internal/content"
	"github.com/rob121/cannon/internal/hooks"
	"github.com/rob121/cannon/internal/models"
	"github.com/rob121/cannon/internal/sites"
	"gorm.io/gorm"
)

const categoriesBase = "/admin/categories"

type categoryListRow struct {
	models.Category
	Depth       int
	CanMoveUp   bool
	CanMoveDown bool
}

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
	case len(parts) == 2 && parts[1] == "move-up":
		h.categoryMoveSort(w, r, parts[0], -1)
	case len(parts) == 2 && parts[1] == "move-down":
		h.categoryMoveSort(w, r, parts[0], 1)
	default:
		id, ok := parseID(parts[0])
		if !ok {
			h.notFound(w, r)
			return
		}
		h.categoryForm(w, r, id)
	}
}

func (h *Handler) categoryList(w http.ResponseWriter, r *http.Request) {
	db, _ := sites.DB(r.Context())
	page := parsePage(r)
	sortParam := strings.TrimSpace(r.URL.Query().Get("sort"))
	useTreeOrder := sortParam == "" || sortParam == "sort"

	data := listPage(r, page, 0, categoriesBase,
		"Organize items into nested categories.",
		"Add Category", map[string]any{"ActiveNav": "categories"})
	order := applyListSort(r, data, map[string]string{
		"name": "name", "slug": "slug", "sort": "sort", "status": "status",
	}, "sort")

	var ordered []models.Category
	if err := db.Order("sort asc, category_id asc").Find(&ordered).Error; err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	sortPos := categorySortPositions(ordered)

	depthByID := make(map[uint]int, len(ordered))
	flat := content.FlattenCategoryOptions(ordered)
	for _, opt := range flat {
		depthByID[opt.CategoryID] = opt.Depth
	}

	var rows []models.Category
	if useTreeOrder {
		rows = make([]models.Category, 0, len(flat))
		for _, opt := range flat {
			rows = append(rows, opt.Category)
		}
		data["Total"] = int64(len(rows))
		data["Page"] = 1
	} else {
		var total int64
		if err := db.Model(&models.Category{}).Count(&total).Error; err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		data["Total"] = total
		if err := db.Model(&models.Category{}).
			Offset((page - 1) * pageSizeFor(r)).
			Limit(pageSizeFor(r)).
			Order(order + ", category_id asc").
			Find(&rows).Error; err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	listRows := make([]categoryListRow, 0, len(rows))
	for _, row := range rows {
		item := categoryListRow{
			Category: row,
			Depth:    depthByID[row.CategoryID],
		}
		if pos, ok := sortPos[row.CategoryID]; ok {
			item.CanMoveUp = pos.canMoveUp
			item.CanMoveDown = pos.canMoveDown
		}
		listRows = append(listRows, item)
	}

	all, _ := content.CategoryTreeAll(r.Context())
	data["Rows"] = listRows
	data["AllCategories"] = all
	data["ListQuery"] = listQueryFromData(data)
	h.render(w, r, "Categories", "admin/categories.html", data)
}

func (h *Handler) categoryForm(w http.ResponseWriter, r *http.Request, id uint) {
	db, _ := sites.DB(r.Context())
	isNew := id == 0
	var row models.Category
	if !isNew {
		if err := db.Preload("Groups").First(&row, id).Error; err != nil {
			h.notFound(w, r)
			return
		}
	} else {
		row.Status = models.StatusActive
		row.Locale = content.LocaleFromContext(r.Context())
		row.InheritSettings = true
		row.InheritPermissions = true
		row.ShowTitle = true
		row.ShowDescription = true
		row.ListColumns = content.DefaultCategoryListColumns
		row.ListPagination = true
		row.ListPageSize = content.DefaultCategoryListPageSize
		if pid, ok := parseID(r.URL.Query().Get("parent_id")); ok {
			row.ParentID = &pid
		}
	}
	allGroups := loadFrontendGroups(db)
	var fieldGroups []models.ContentFieldGroup
	db.Order("name asc").Find(&fieldGroups)
	allCats, _ := content.CategoryTreeAll(r.Context())

	if r.Method == http.MethodPost {
		if err := r.ParseForm(); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		row.Name = formString(r, "name")
		row.Locale = formString(r, "locale")
		content.NormalizeCategoryLocale(r.Context(), &row)
		row.ParentID = formUintPtr(r, "parent_id")
		slugCtx := content.WithLocale(r.Context(), row.Locale)
		var slugErr error
		row.Slug, slugErr = content.ResolveCategorySlug(slugCtx, row.Name, formString(r, "slug"), row.ParentID, row.CategoryID)
		if slugErr != nil {
			h.renderCategoryForm(w, r, row, allGroups, fieldGroups, allCats, isNew, slugErr.Error())
			return
		}
		row.Description = r.FormValue("description")
		row.Image = formString(r, "image")
		row.Template = formString(r, "template")
		row.Sort = formInt(r, "sort", 0)
		row.Status = formStatus(r)
		row.InheritSettings = formBool(r, "inherit_settings")
		row.ShowTitle = formBool(r, "show_title")
		row.ShowDescription = formBool(r, "show_description")
		row.ListColumns = content.NormalizeCategoryListColumns(formInt(r, "list_columns", content.DefaultCategoryListColumns))
		row.ListPagination = formBool(r, "list_pagination")
		row.ListPageSize = content.NormalizeCategoryListPageSize(formInt(r, "list_page_size", content.DefaultCategoryListPageSize))
		row.FieldGroupID = formUintPtr(r, "field_group_id")
		beforeArgs := map[string]any{
			"category": row,
			"is_new":   isNew,
			"form":     r.Form,
		}
		if _, err := hooks.Fire(r.Context(), hooks.OnCategoryBeforeSave, beforeArgs); err != nil {
			h.renderCategoryForm(w, r, row, allGroups, fieldGroups, allCats, isNew, err.Error())
			return
		}
		var saveErr error
		if isNew {
			saveErr = db.Select(
				"ParentID", "Locale", "Name", "Slug", "Description", "Image", "Template",
				"FieldGroupID", "InheritSettings", "InheritPermissions", "ShowTitle", "ShowDescription", "ListColumns", "ListPagination", "ListPageSize", "Sort", "Status",
			).Create(&row).Error
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
		_, _ = hooks.Fire(r.Context(), hooks.OnCategoryAfterSave, map[string]any{
			"category": row,
			"is_new":   isNew,
		})
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
	db, _ := sites.DB(r.Context())
	data := formData(map[string]any{
		"ActiveNav":       "categories",
		"Row":             row,
		"IsNew":           isNew,
		"BasePath":        categoriesBase,
		"Subtitle":        subtitle,
		"AllGroups":       allGroups,
		"SelectedIDs":     defaultGroupSelectedIDs(db, row.Groups, isNew),
		"FieldGroups":     fieldGroups,
		"AllCategories":   content.CategoryParentOptions(allCats, row.CategoryID),
		"ContentLocales": adminContentLocales(r),
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
		h.notFound(w, r)
		return
	}
	db, _ := sites.DB(r.Context())
	var row models.Category
	if err := db.First(&row, id).Error; err != nil {
		h.notFound(w, r)
		return
	}
	if _, err := hooks.Fire(r.Context(), hooks.OnCategoryBeforeDelete, map[string]any{
		"category_id": id,
		"category":    row,
	}); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	db.Exec("DELETE FROM category_groups WHERE category_category_id = ?", id)
	db.Exec("DELETE FROM category_create_groups WHERE category_category_id = ?", id)
	db.Exec("DELETE FROM category_edit_groups WHERE category_category_id = ?", id)
	db.Exec("DELETE FROM category_publish_groups WHERE category_category_id = ?", id)
	if err := db.Delete(&models.Category{}, id).Error; err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	redirectList(w, r, categoriesBase)
}

func (h *Handler) categoryToggleStatus(w http.ResponseWriter, r *http.Request, idStr string) {
	h.postToggleModel(w, r, idStr, &models.Category{}, categoriesBase, nil)
}

type categorySortPosition struct {
	canMoveUp   bool
	canMoveDown bool
}

func categorySiblingKey(row models.Category) uint {
	if row.ParentID == nil || *row.ParentID == 0 {
		return 0
	}
	return *row.ParentID
}

func categorySortPositions(categories []models.Category) map[uint]categorySortPosition {
	byParent := make(map[uint][]models.Category)
	for _, row := range categories {
		key := categorySiblingKey(row)
		byParent[key] = append(byParent[key], row)
	}
	out := make(map[uint]categorySortPosition, len(categories))
	for _, group := range byParent {
		sort.Slice(group, func(i, j int) bool {
			if group[i].Sort != group[j].Sort {
				return group[i].Sort < group[j].Sort
			}
			return group[i].CategoryID < group[j].CategoryID
		})
		last := len(group) - 1
		for i, row := range group {
			out[row.CategoryID] = categorySortPosition{
				canMoveUp:   i > 0,
				canMoveDown: i < last,
			}
		}
	}
	return out
}

func (h *Handler) categoryMoveSort(w http.ResponseWriter, r *http.Request, idStr string, direction int) {
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
	if err := categoryReorder(db, id, direction); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	redirectList(w, r, categoriesBase+listRedirectQuery(r))
}

func categoryReorder(db *gorm.DB, id uint, direction int) error {
	if direction == 0 {
		return nil
	}
	var row models.Category
	if err := db.First(&row, id).Error; err != nil {
		return err
	}
	var siblings []models.Category
	q := db.Model(&models.Category{})
	if row.ParentID == nil || *row.ParentID == 0 {
		q = q.Where("parent_id IS NULL OR parent_id = 0")
	} else {
		q = q.Where("parent_id = ?", *row.ParentID)
	}
	if err := q.Order("sort asc, category_id asc").Find(&siblings).Error; err != nil {
		return err
	}
	idx := -1
	for i, item := range siblings {
		if item.CategoryID == id {
			idx = i
			break
		}
	}
	if idx < 0 {
		return gorm.ErrRecordNotFound
	}
	target := idx + direction
	if target < 0 || target >= len(siblings) {
		return nil
	}
	siblings[idx], siblings[target] = siblings[target], siblings[idx]
	for i, item := range siblings {
		if item.Sort == i {
			continue
		}
		if err := db.Model(&models.Category{}).Where("category_id = ?", item.CategoryID).Update("sort", i).Error; err != nil {
			return err
		}
	}
	return nil
}
