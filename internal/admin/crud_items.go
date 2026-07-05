package admin

import (
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/rob121/cannon/internal/content"
	"github.com/rob121/cannon/internal/hooks"
	"github.com/rob121/cannon/internal/models"
	"github.com/rob121/cannon/internal/sites"
	"gorm.io/gorm"
)

const itemsBase = "/admin/items"

type itemListRow struct {
	models.Item
	CategoryName        string
	AuthorName          string
	CanMoveUp           bool
	CanMoveDown         bool
	CanFeaturedMoveUp   bool
	CanFeaturedMoveDown bool
}

func (h *Handler) items(w http.ResponseWriter, r *http.Request, path string) {
	parts := pathParts("/items", path)
	switch {
	case len(parts) == 0:
		h.itemList(w, r)
	case parts[0] == "new":
		h.itemForm(w, r, 0)
	case parts[0] == "bulk":
		h.itemBulk(w, r)
	case len(parts) == 2 && parts[1] == "revisions":
		h.itemRevisions(w, r, parts[0])
	case len(parts) == 4 && parts[1] == "revisions" && parts[3] == "restore":
		h.itemRevisionRestore(w, r, parts[0], parts[2])
	case len(parts) == 2 && parts[1] == "preview-token":
		h.itemPreviewToken(w, r, parts[0])
	case len(parts) == 2 && parts[1] == "submit-review":
		h.itemSubmitReview(w, r, parts[0])
	case len(parts) == 2 && parts[1] == "clone":
		h.itemClone(w, r, parts[0])
	case len(parts) == 2 && parts[1] == "delete":
		h.itemDelete(w, r, parts[0])
	case len(parts) == 2 && parts[1] == "move-up":
		h.itemMoveSort(w, r, parts[0], -1)
	case len(parts) == 2 && parts[1] == "move-down":
		h.itemMoveSort(w, r, parts[0], 1)
	case len(parts) == 2 && parts[1] == "move-featured-up":
		h.itemMoveFeaturedSort(w, r, parts[0], -1)
	case len(parts) == 2 && parts[1] == "move-featured-down":
		h.itemMoveFeaturedSort(w, r, parts[0], 1)
	default:
		id, ok := parseID(parts[0])
		if !ok {
			h.notFound(w, r)
			return
		}
		h.itemForm(w, r, id)
	}
}

func itemListExcludeTrashed(q *gorm.DB) *gorm.DB {
	return q.Where("status <> ?", models.ItemStatusTrashed)
}

func itemListFilters(db *gorm.DB, statusFilter, categoryFilter, featuredFilter, query string) *gorm.DB {
	q := itemListExcludeTrashed(db.Model(&models.Item{}))
	if statusFilter != "" && statusFilter != string(models.ItemStatusTrashed) {
		q = q.Where("status = ?", statusFilter)
	}
	if categoryFilter != "" {
		if id, ok := parseID(categoryFilter); ok {
			q = q.Where("category_id = ?", id)
		}
	}
	if featuredFilter == "1" {
		q = q.Where("featured = ?", true)
	} else if featuredFilter == "0" {
		q = q.Where("featured = ?", false)
	}
	if query != "" {
		like := "%" + query + "%"
		q = q.Where("title LIKE ? OR slug LIKE ?", like, like)
	}
	return q
}

func (h *Handler) itemList(w http.ResponseWriter, r *http.Request) {
	db, _ := sites.DB(r.Context())
	page := parsePage(r)
	statusFilter := r.URL.Query().Get("status")
	if statusFilter == string(models.ItemStatusTrashed) {
		statusFilter = ""
	}
	categoryFilter := r.URL.Query().Get("category")
	featuredFilter := r.URL.Query().Get("featured")
	query := r.URL.Query().Get("q")

	q := itemListFilters(db, statusFilter, categoryFilter, featuredFilter, query)
	var total int64
	q.Count(&total)

	data := listPage(r, page, total, itemsBase,
		"Create and manage structured content items.",
		"Add Item", map[string]any{"ActiveNav": "items"})
	order := applyListSort(r, data, map[string]string{
		"title": "title", "status": "status", "sort": "sort", "featured": "featured", "featured_sort": "featured_sort",
	}, "sort")
	if featuredFilter == "1" && strings.TrimSpace(r.URL.Query().Get("sort")) == "" {
		data["Sort"] = "featured_sort"
		data["Dir"] = "asc"
		order = "featured_sort asc"
	}

	listQ := itemListFilters(db, statusFilter, categoryFilter, featuredFilter, query)
	var rows []models.Item
	listQ.Preload("Category").Preload("Author").
		Offset((page - 1) * pageSizeFor(r)).Limit(pageSizeFor(r)).Order(order).Find(&rows)

	var ordered []models.Item
	itemListExcludeTrashed(db).Select("item_id", "category_id", "featured", "featured_sort", "sort").Order("sort asc, item_id asc").Find(&ordered)
	sortPos := itemSortPositions(ordered)

	var featuredOrdered []models.Item
	db.Select("item_id", "featured_sort").Where("featured = ?", true).Order("featured_sort asc, item_id asc").Find(&featuredOrdered)
	featuredSortPos := itemFeaturedSortPositions(featuredOrdered)

	listRows := make([]itemListRow, 0, len(rows))
	for _, row := range rows {
		lr := itemListRow{Item: row}
		if row.Category != nil {
			lr.CategoryName = row.Category.Name
		}
		if row.Author != nil {
			lr.AuthorName = row.Author.Username
		}
		if pos, ok := sortPos[row.ItemID]; ok {
			lr.CanMoveUp = pos.canMoveUp
			lr.CanMoveDown = pos.canMoveDown
		}
		if row.Featured {
			if pos, ok := featuredSortPos[row.ItemID]; ok {
				lr.CanFeaturedMoveUp = pos.canMoveUp
				lr.CanFeaturedMoveDown = pos.canMoveDown
			}
		}
		listRows = append(listRows, lr)
	}

	categories, _ := content.CategoryTreeAll(r.Context())
	var allTags []models.Tag
	db.Order("name asc").Find(&allTags)
	var categoryFilterID uint
	if categoryFilter != "" {
		if id, ok := parseID(categoryFilter); ok {
			categoryFilterID = id
		}
	}
	data["Rows"] = listRows
	data["StatusFilter"] = statusFilter
	data["CategoryFilter"] = categoryFilter
	data["FeaturedFilter"] = featuredFilter
	data["CategoryFilterID"] = categoryFilterID
	data["SearchQuery"] = query
	data["Categories"] = content.FlattenCategoryOptions(categories)
	data["AllTags"] = allTags
	data["ListQuery"] = listQueryFromData(data)
	h.render(w, r, "Items", "admin/items.html", data)
}

func (h *Handler) itemForm(w http.ResponseWriter, r *http.Request, id uint) {
	db, _ := sites.DB(r.Context())
	isNew := id == 0
	var row models.Item
	if !isNew {
		if err := db.Preload("Groups").Preload("Tags").Preload("Category").Preload("Author").First(&row, id).Error; err != nil {
			h.notFound(w, r, "This item could not be found.")
			return
		}
	} else {
		row.Status = models.ItemStatusDraft
		row.Locale = content.LocaleFromContext(r.Context())
	}
	allGroups := loadFrontendGroups(db)
	var allTags []models.Tag
	db.Order("name asc").Find(&allTags)
	categories, _ := content.CategoriesByLocale(r.Context(), row.Locale)
	var users []models.User
	db.Where("status = ?", models.StatusActive).Order("username asc").Find(&users)
	customFields, fieldValues := itemCustomFields(r, db, &row)

	if r.Method == http.MethodPost {
		if err := r.ParseForm(); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if err := h.saveItemFromForm(r, db, &row, isNew); err != nil {
			customFields, fieldValues = itemCustomFields(r, db, &row)
			h.renderItemForm(w, r, db, row, allGroups, allTags, categories, users, customFields, fieldValues, isNew, err.Error())
			return
		}
		redirectList(w, r, itemsBase+listRedirectQuery(r))
		return
	}
	h.renderItemForm(w, r, db, row, allGroups, allTags, categories, users, customFields, fieldValues, isNew, "")
}

func (h *Handler) saveItemFromForm(r *http.Request, db *gorm.DB, row *models.Item, isNew bool) error {
	row.Title = formString(r, "title")
	row.Locale = formString(r, "locale")
	content.NormalizeItemLocale(r.Context(), row)
	row.Slug = formString(r, "slug")
	slugCtx := content.WithLocale(r.Context(), row.Locale)
	if row.Slug == "" {
		slug, err := content.UniqueItemSlug(slugCtx, row.Title, row.ItemID)
		if err != nil {
			return err
		}
		row.Slug = slug
	}
	row.Intro = r.FormValue("intro")
	row.Body = r.FormValue("body")
	row.Status = formItemStatus(r)
	wasFeatured := false
	if !isNew {
		var prev models.Item
		if err := db.Select("featured", "featured_sort").First(&prev, row.ItemID).Error; err == nil {
			wasFeatured = prev.Featured
		}
	}
	row.Featured = formBool(r, "featured")
	if row.Featured {
		if wasFeatured && !isNew {
			var prev models.Item
			if err := db.Select("featured_sort").First(&prev, row.ItemID).Error; err == nil {
				row.FeaturedSort = prev.FeaturedSort
			}
		} else {
			var count int64
			q := db.Model(&models.Item{}).Where("featured = ?", true)
			if !isNew {
				q = q.Where("item_id <> ?", row.ItemID)
			}
			_ = q.Count(&count).Error
			row.FeaturedSort = int(count)
		}
	}
	row.Sort = formInt(r, "sort", 0)
	row.Image = formString(r, "image")
	galleryJSON, embedsJSON, attachmentsJSON := content.ItemMediaFromForm(r)
	row.GalleryJSON = galleryJSON
	row.EmbedJSON = embedsJSON
	row.AttachmentsJSON = attachmentsJSON
	row.MetaTitle = formString(r, "meta_title")
	row.MetaDescription = r.FormValue("meta_description")
	row.MetaKeywords = formString(r, "meta_keywords")
	row.CanonicalURL = formString(r, "canonical_url")
	row.CategoryID = formUintPtr(r, "category_id")
	row.AuthorID = formUintPtr(r, "author_id")
	row.PublishStart = formTimePtr(r, "publish_start")
	row.PublishEnd = formTimePtr(r, "publish_end")

	var cat *models.Category
	if row.CategoryID != nil && *row.CategoryID > 0 {
		var c models.Category
		if err := db.First(&c, *row.CategoryID).Error; err == nil {
			cat = &c
		}
	}
	customFields, _ := content.FieldsForCategory(r.Context(), cat)
	if err := content.ValidateRequiredCustomFields(customFields, r); err != nil {
		return err
	}

	beforeArgs := map[string]any{
		"item":   row,
		"is_new": isNew,
		"form":   r.Form,
	}
	if _, err := hooks.Fire(r.Context(), hooks.OnItemBeforeSave, beforeArgs); err != nil {
		return err
	}

	var saveErr error
	if !isNew {
		editorID, editorName := currentEditor(r)
		if err := content.CreateItemRevision(r.Context(), db, row.ItemID, editorID, editorName); err != nil {
			return err
		}
		saveErr = db.Save(row).Error
	} else {
		saveErr = db.Create(row).Error
	}
	if saveErr != nil {
		return saveErr
	}
	if err := replaceFormGroups(db, row, r); err != nil {
		return err
	}
	if err := replaceItemTags(db, row, r); err != nil {
		return err
	}
	if err := saveItemFieldValues(db, row, customFields, r); err != nil {
		return err
	}
	if linkID, ok := parseID(formString(r, "translation_of_item_id")); ok {
		if err := content.LinkTranslation(r.Context(), row.ItemID, linkID); err != nil {
			return err
		}
	}
	if err := content.UpsertSearchIndex(r.Context(), db, row); err != nil {
		return err
	}
	afterArgs := map[string]any{"item_id": row.ItemID, "item": row}
	_, err := hooks.Fire(r.Context(), hooks.OnItemAfterSave, afterArgs)
	return err
}

func replaceItemTags(db *gorm.DB, row *models.Item, r *http.Request) error {
	var tags []models.Tag
	for _, s := range r.Form["tag_ids"] {
		if id, ok := parseID(s); ok {
			tags = append(tags, models.Tag{TagID: id})
		}
	}
	return db.Model(row).Association("Tags").Replace(tags)
}

func saveItemFieldValues(db *gorm.DB, row *models.Item, fields []models.ContentField, r *http.Request) error {
	return content.SaveItemFieldValues(db, row.ItemID, fields, r)
}

func itemCustomFields(r *http.Request, db *gorm.DB, row *models.Item) ([]models.ContentField, map[uint]string) {
	values := map[uint]string{}
	var cat *models.Category
	if row.CategoryID != nil && *row.CategoryID > 0 {
		var c models.Category
		if db.First(&c, *row.CategoryID).Error == nil {
			cat = &c
		}
	}
	fields, _ := content.FieldsForCategory(r.Context(), cat)
	if row.ItemID > 0 {
		var rows []models.ItemFieldValue
		db.Where("item_id = ?", row.ItemID).Find(&rows)
		for _, v := range rows {
			values[v.FieldID] = v.Value
		}
	}
	return fields, values
}

func (h *Handler) renderItemForm(w http.ResponseWriter, r *http.Request, db *gorm.DB, row models.Item, allGroups []models.Group, allTags []models.Tag, categories []models.Category, users []models.User, customFields []models.ContentField, fieldValues map[uint]string, isNew bool, errMsg string) {
	title := "Add Item"
	subtitle := "Create a new content item."
	if !isNew {
		title = "Edit Item"
		subtitle = "Update item content, metadata, and visibility."
	}
	tagIDs := make([]uint, 0, len(row.Tags))
	for _, t := range row.Tags {
		tagIDs = append(tagIDs, t.TagID)
	}
	data := formData(map[string]any{
		"ActiveNav":     "items",
		"Row":           row,
		"IsNew":         isNew,
		"BasePath":      itemsBase,
		"Subtitle":      subtitle,
		"AllGroups":     allGroups,
		"SelectedIDs":   defaultGroupSelectedIDs(db, row.Groups, isNew),
		"AllTags":       allTags,
		"SelectedTagIDs": tagIDs,
		"Categories":    content.FlattenCategoryOptions(categories),
		"Users":         users,
		"CustomFields":  customFields,
		"FieldValues":   fieldValues,
		"Gallery":       content.ParseGalleryJSON(row.GalleryJSON),
		"Embeds":        content.ParseEmbedsJSON(row.EmbedJSON),
		"Attachments":   content.ParseAttachmentsJSON(row.AttachmentsJSON),
		"PreviewURL":    itemPreviewURL(r, row),
		"ShowPreview":   r.URL.Query().Get("preview") == "1",
		"RevisionsURL":  fmt.Sprintf("%s/%d/revisions", itemsBase, row.ItemID),
		"ContentLocales": adminContentLocales(r),
		"Translations":  itemTranslations(r, row),
	})
	if errMsg != "" {
		data["Error"] = errMsg
	}
	h.render(w, r, title, "admin/items_form.html", data)
}

func itemPreviewURL(r *http.Request, row models.Item) string {
	if row.ItemID == 0 || row.PreviewToken == "" || row.PreviewExpiresAt == nil {
		return ""
	}
	if time.Now().After(*row.PreviewExpiresAt) {
		return ""
	}
	path := content.PreviewURL(r.Context(), row.PreviewToken)
	site, err := sites.FromContext(r.Context())
	if err != nil || strings.TrimSpace(site.Host) == "" {
		return path
	}
	return absoluteSiteURL(site.Host, path)
}

func (h *Handler) itemClone(w http.ResponseWriter, r *http.Request, idStr string) {
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
	clone, err := content.CloneItem(r.Context(), db, id)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			h.notFound(w, r, "This item could not be found.")
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	redirectList(w, r, fmt.Sprintf("%s/%d", itemsBase, clone.ItemID))
}

func (h *Handler) itemDelete(w http.ResponseWriter, r *http.Request, idStr string) {
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
	deleteItemPermanent(db, id)
	redirectList(w, r, itemsBase+listRedirectQuery(r))
}

func (h *Handler) itemBulk(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	action := formString(r, "bulk_action")
	db, _ := sites.DB(r.Context())
	for _, idStr := range r.Form["item_ids"] {
		id, ok := parseID(idStr)
		if !ok {
			continue
		}
		switch action {
		case "publish":
			db.Model(&models.Item{}).Where("item_id = ?", id).Update("status", models.ItemStatusPublished)
		case "draft":
			db.Model(&models.Item{}).Where("item_id = ?", id).Update("status", models.ItemStatusDraft)
		case "archive":
			db.Model(&models.Item{}).Where("item_id = ?", id).Update("status", models.ItemStatusArchived)
		case "trash":
			db.Model(&models.Item{}).Where("item_id = ?", id).Update("status", models.ItemStatusTrashed)
		case "restore":
			db.Model(&models.Item{}).Where("item_id = ? AND status = ?", id, models.ItemStatusTrashed).
				Update("status", models.ItemStatusDraft)
		case "approve":
			db.Model(&models.Item{}).Where("item_id = ? AND status = ?", id, models.ItemStatusPending).
				Update("status", models.ItemStatusPublished)
		case "reject":
			db.Model(&models.Item{}).Where("item_id = ? AND status = ?", id, models.ItemStatusPending).
				Update("status", models.ItemStatusDraft)
		case "delete":
			deleteItemPermanent(db, id)
		case "assign_category":
			catID := formUintPtr(r, "bulk_category_id")
			if catID != nil {
				db.Model(&models.Item{}).Where("item_id = ?", id).Update("category_id", *catID)
			}
		case "assign_tags":
			var item models.Item
			if db.First(&item, id).Error != nil {
				continue
			}
			var tags []models.Tag
			for _, s := range r.Form["bulk_tag_ids"] {
				if tid, ok := parseID(s); ok {
					tags = append(tags, models.Tag{TagID: tid})
				}
			}
			if len(tags) > 0 {
				_ = db.Model(&item).Association("Tags").Replace(tags)
			}
		}
	}
	redirectList(w, r, itemsBase+listRedirectQuery(r))
}

type itemSortPosition struct {
	canMoveUp   bool
	canMoveDown bool
}

func itemCategoryKey(categoryID *uint) uint {
	if categoryID == nil || *categoryID == 0 {
		return 0
	}
	return *categoryID
}

func itemSortPositions(items []models.Item) map[uint]itemSortPosition {
	byCategory := make(map[uint][]models.Item)
	for _, row := range items {
		key := itemCategoryKey(row.CategoryID)
		byCategory[key] = append(byCategory[key], row)
	}
	out := make(map[uint]itemSortPosition, len(items))
	for _, group := range byCategory {
		sort.Slice(group, func(i, j int) bool {
			if group[i].Sort != group[j].Sort {
				return group[i].Sort < group[j].Sort
			}
			return group[i].ItemID < group[j].ItemID
		})
		last := len(group) - 1
		for i, row := range group {
			out[row.ItemID] = itemSortPosition{
				canMoveUp:   i > 0,
				canMoveDown: i < last,
			}
		}
	}
	return out
}

func (h *Handler) itemMoveSort(w http.ResponseWriter, r *http.Request, idStr string, direction int) {
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
	if err := itemReorder(db, id, direction); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	redirectList(w, r, itemsBase+listRedirectQuery(r))
}

func itemReorder(db *gorm.DB, id uint, direction int) error {
	if direction == 0 {
		return nil
	}
	var row models.Item
	if err := db.First(&row, id).Error; err != nil {
		return err
	}
	var siblings []models.Item
	q := db.Model(&models.Item{})
	if row.CategoryID == nil || *row.CategoryID == 0 {
		q = q.Where("category_id IS NULL OR category_id = 0")
	} else {
		q = q.Where("category_id = ?", *row.CategoryID)
	}
	if err := q.Order("sort asc, item_id asc").Find(&siblings).Error; err != nil {
		return err
	}
	idx := -1
	for i, item := range siblings {
		if item.ItemID == id {
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
		if err := db.Model(&models.Item{}).Where("item_id = ?", item.ItemID).Update("sort", i).Error; err != nil {
			return err
		}
	}
	return nil
}

func itemFeaturedSortPositions(items []models.Item) map[uint]itemSortPosition {
	out := make(map[uint]itemSortPosition, len(items))
	sort.Slice(items, func(i, j int) bool {
		if items[i].FeaturedSort != items[j].FeaturedSort {
			return items[i].FeaturedSort < items[j].FeaturedSort
		}
		return items[i].ItemID < items[j].ItemID
	})
	last := len(items) - 1
	for i, row := range items {
		out[row.ItemID] = itemSortPosition{
			canMoveUp:   i > 0,
			canMoveDown: i < last,
		}
	}
	return out
}

func (h *Handler) itemMoveFeaturedSort(w http.ResponseWriter, r *http.Request, idStr string, direction int) {
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
	if err := itemFeaturedReorder(db, id, direction); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	redirectList(w, r, itemsBase+listRedirectQuery(r))
}

func itemFeaturedReorder(db *gorm.DB, id uint, direction int) error {
	if direction == 0 {
		return nil
	}
	var row models.Item
	if err := db.First(&row, id).Error; err != nil {
		return err
	}
	if !row.Featured {
		return nil
	}
	var siblings []models.Item
	if err := db.Model(&models.Item{}).Where("featured = ?", true).
		Order("featured_sort asc, item_id asc").Find(&siblings).Error; err != nil {
		return err
	}
	idx := -1
	for i, item := range siblings {
		if item.ItemID == id {
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
		if item.FeaturedSort == i {
			continue
		}
		if err := db.Model(&models.Item{}).Where("item_id = ?", item.ItemID).Update("featured_sort", i).Error; err != nil {
			return err
		}
	}
	return nil
}

func adminContentLocales(r *http.Request) []string {
	_, _, locales, err := content.LocaleConfig(r.Context())
	if err != nil || len(locales) == 0 {
		return []string{"en-US"}
	}
	return locales
}

func itemTranslations(r *http.Request, row models.Item) []models.Item {
	if row.ItemID == 0 {
		return nil
	}
	rows, err := content.ItemTranslations(r.Context(), &row)
	if err != nil {
		return nil
	}
	return rows
}
