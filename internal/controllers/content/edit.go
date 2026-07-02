package content

import (
	"net/http"
	"strconv"
	"strings"

	cms "github.com/rob121/cannon/internal/content"
	"github.com/rob121/cannon/internal/controllers"
	"github.com/rob121/cannon/internal/hooks"
	"github.com/rob121/cannon/internal/models"
	"github.com/rob121/cannon/internal/sites"
	"gorm.io/gorm"
)

func (c *Controller) handleEditNew(ctx *controllers.Context) controllers.Result {
	user, err := ctx.CurrentUser()
	if err != nil {
		return controllers.Error(http.StatusUnauthorized, "login required")
	}
	ok, err := cms.CanCreateItem(ctx.GoContext(), user.UserID)
	if err != nil {
		return controllers.Error(http.StatusInternalServerError, err.Error())
	}
	if !ok {
		return controllers.Error(http.StatusForbidden, "permission denied")
	}
	canPublish, _ := cms.CanPublishItem(ctx.GoContext(), user.UserID)
	item := models.Item{Status: models.ItemStatusDraft, AuthorID: &user.UserID}
	if ctx.Request.Method == http.MethodPost {
		if err := ctx.Request.ParseForm(); err != nil {
			return controllers.Error(http.StatusBadRequest, err.Error())
		}
		if err := saveFrontendItem(ctx, &item, true, canPublish); err != nil {
			return renderEditForm(ctx, item, nil, nil, true, canPublish, err.Error())
		}
		return controllers.Redirect(http.StatusSeeOther, cms.ItemURL(item.Slug))
	}
	return renderEditForm(ctx, item, nil, nil, true, canPublish, "")
}

func (c *Controller) handleEdit(ctx *controllers.Context) controllers.Result {
	slug := strings.Trim(ctx.PathSuffix(), "/")
	if slug == "" {
		return controllers.Error(http.StatusNotFound, "item not found")
	}
	user, err := ctx.CurrentUser()
	if err != nil {
		return controllers.Error(http.StatusUnauthorized, "login required")
	}
	db, err := sites.DB(ctx.GoContext())
	if err != nil {
		return controllers.Error(http.StatusInternalServerError, err.Error())
	}
	var item models.Item
	if err := db.Preload("Tags").Preload("Category").First(&item, "slug = ?", slug).Error; err != nil {
		return controllers.Error(http.StatusNotFound, "item not found")
	}
	ok, err := cms.CanEditItem(ctx.GoContext(), user.UserID, &item)
	if err != nil {
		return controllers.Error(http.StatusInternalServerError, err.Error())
	}
	if !ok {
		return controllers.Error(http.StatusForbidden, "permission denied")
	}
	canPublish, _ := cms.CanPublishItem(ctx.GoContext(), user.UserID)
	categories, _ := cms.CategoryTree(ctx.GoContext())
	tags, _ := cms.ListTags(ctx.GoContext())
	fields, fieldValues := frontendCustomFields(ctx, db, &item)

	if ctx.Request.Method == http.MethodPost {
		if err := ctx.Request.ParseForm(); err != nil {
			return controllers.Error(http.StatusBadRequest, err.Error())
		}
		if err := saveFrontendItem(ctx, &item, false, canPublish); err != nil {
			return renderEditForm(ctx, item, categories, tags, false, canPublish, err.Error())
		}
		return controllers.Redirect(http.StatusSeeOther, cms.ItemURL(item.Slug))
	}
	data := editFormData(item, categories, tags, fields, fieldValues, false, canPublish, "")
	return controllers.HTMLPage("Edit Item", "default/controllers/content/edit.html", data)
}

func saveFrontendItem(ctx *controllers.Context, item *models.Item, isNew, canPublish bool) error {
	db, err := sites.DB(ctx.GoContext())
	if err != nil {
		return err
	}
	item.Title = strings.TrimSpace(ctx.Request.FormValue("title"))
	item.Intro = ctx.Request.FormValue("intro")
	item.Body = ctx.Request.FormValue("body")
	item.Slug = strings.TrimSpace(ctx.Request.FormValue("slug"))
	if item.Slug == "" {
		slug, err := cms.UniqueItemSlug(ctx.GoContext(), item.Title, item.ItemID)
		if err != nil {
			return err
		}
		item.Slug = slug
	}
	item.Image = strings.TrimSpace(ctx.Request.FormValue("image"))
	item.CategoryID = formUintPtr(ctx.Request, "category_id")
	status := strings.TrimSpace(ctx.Request.FormValue("status"))
	if canPublish && status == string(models.ItemStatusPublished) {
		item.Status = models.ItemStatusPublished
	} else {
		item.Status = models.ItemStatusDraft
	}
	beforeArgs := map[string]any{"item": item, "is_new": isNew, "form": ctx.Request.Form}
	if _, err := hooks.Fire(ctx.GoContext(), hooks.OnItemBeforeSave, beforeArgs); err != nil {
		return err
	}
	if isNew {
		if err := db.Create(item).Error; err != nil {
			return err
		}
	} else if err := db.Save(item).Error; err != nil {
		return err
	}
	if err := replaceFrontendTags(db, item, ctx.Request); err != nil {
		return err
	}
	if err := saveFrontendFieldValues(db, item, ctx.Request); err != nil {
		return err
	}
	afterArgs := map[string]any{"item_id": item.ItemID, "item": item}
	_, err = hooks.Fire(ctx.GoContext(), hooks.OnItemAfterSave, afterArgs)
	return err
}

func replaceFrontendTags(db *gorm.DB, item *models.Item, r *http.Request) error {
	var tags []models.Tag
	for _, s := range r.Form["tag_ids"] {
		if id, err := strconv.ParseUint(s, 10, 64); err == nil && id > 0 {
			tags = append(tags, models.Tag{TagID: uint(id)})
		}
	}
	return db.Model(item).Association("Tags").Replace(tags)
}

func saveFrontendFieldValues(db *gorm.DB, item *models.Item, r *http.Request) error {
	for key, vals := range r.Form {
		if len(key) < 6 || key[:6] != "field_" {
			continue
		}
		id, err := strconv.ParseUint(key[6:], 10, 64)
		if err != nil || id == 0 {
			continue
		}
		value := ""
		if len(vals) > 0 {
			value = vals[0]
		}
		fieldID := uint(id)
		var existing models.ItemFieldValue
		err = db.Where("item_id = ? AND field_id = ?", item.ItemID, fieldID).First(&existing).Error
		if err == gorm.ErrRecordNotFound {
			if err := db.Create(&models.ItemFieldValue{ItemID: item.ItemID, FieldID: fieldID, Value: value}).Error; err != nil {
				return err
			}
			continue
		}
		if err != nil {
			return err
		}
		existing.Value = value
		if err := db.Save(&existing).Error; err != nil {
			return err
		}
	}
	return nil
}

func frontendCustomFields(ctx *controllers.Context, db *gorm.DB, item *models.Item) ([]models.ContentField, map[uint]string) {
	values := map[uint]string{}
	var cat *models.Category
	if item.CategoryID != nil && *item.CategoryID > 0 {
		var c models.Category
		if db.First(&c, *item.CategoryID).Error == nil {
			cat = &c
		}
	}
	fields, _ := cms.FieldsForCategory(ctx.GoContext(), cat)
	if item.ItemID > 0 {
		var rows []models.ItemFieldValue
		db.Where("item_id = ?", item.ItemID).Find(&rows)
		for _, v := range rows {
			values[v.FieldID] = v.Value
		}
	}
	return fields, values
}

func renderEditForm(ctx *controllers.Context, item models.Item, categories []models.Category, tags []models.Tag, isNew, canPublish bool, errMsg string) controllers.Result {
	if categories == nil {
		categories, _ = cms.CategoryTree(ctx.GoContext())
	}
	if tags == nil {
		tags, _ = cms.ListTags(ctx.GoContext())
	}
	db, _ := sites.DB(ctx.GoContext())
	fields, fieldValues := frontendCustomFields(ctx, db, &item)
	data := editFormData(item, categories, tags, fields, fieldValues, isNew, canPublish, errMsg)
	title := "Create Item"
	if !isNew {
		title = "Edit Item"
	}
	return controllers.HTMLPage(title, "default/controllers/content/edit.html", data)
}

func editFormData(item models.Item, categories []models.Category, tags []models.Tag, fields []models.ContentField, fieldValues map[uint]string, isNew, canPublish bool, errMsg string) map[string]any {
	tagIDs := make([]uint, 0, len(item.Tags))
	for _, t := range item.Tags {
		tagIDs = append(tagIDs, t.TagID)
	}
	var selectedCategoryID uint
	if item.CategoryID != nil {
		selectedCategoryID = *item.CategoryID
	}
	data := map[string]any{
		"Item":               item,
		"IsNew":              isNew,
		"CanPublish":         canPublish,
		"Categories":         categories,
		"Tags":               tags,
		"SelectedTagIDs":     tagIDs,
		"SelectedCategoryID": selectedCategoryID,
		"CustomFields":       fields,
		"FieldValues":        fieldValues,
	}
	if errMsg != "" {
		data["Error"] = errMsg
	}
	return data
}

func formUintPtr(r *http.Request, key string) *uint {
	v, err := strconv.ParseUint(strings.TrimSpace(r.FormValue(key)), 10, 64)
	if err != nil || v == 0 {
		return nil
	}
	id := uint(v)
	return &id
}
