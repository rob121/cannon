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
	canPublish, _ := cms.CanPublishItem(ctx.GoContext(), user.UserID, nil)
	item := models.Item{Status: models.ItemStatusDraft, AuthorID: &user.UserID}
	if catID := formUintPtr(ctx.Request, "category_id"); catID != nil {
		item.CategoryID = catID
		if ok, err := cms.CanCreateItemInCategory(ctx.GoContext(), user.UserID, catID); err != nil {
			return controllers.Error(http.StatusInternalServerError, err.Error())
		} else if !ok {
			return controllers.Error(http.StatusForbidden, "permission denied")
		}
		canPublish, _ = cms.CanPublishItem(ctx.GoContext(), user.UserID, catID)
	}
	if ctx.Request.Method == http.MethodPost {
		if err := ctx.Request.ParseForm(); err != nil {
			return controllers.Error(http.StatusBadRequest, err.Error())
		}
		categoryID := formUintPtr(ctx.Request, "category_id")
		ok, err := cms.CanCreateItemInCategory(ctx.GoContext(), user.UserID, categoryID)
		if err != nil {
			return controllers.Error(http.StatusInternalServerError, err.Error())
		}
		if !ok {
			return controllers.Error(http.StatusForbidden, "permission denied")
		}
		if err := saveFrontendItem(ctx, &item, true, canPublish); err != nil {
			return renderEditForm(ctx, item, nil, nil, true, canPublish, err.Error())
		}
		return controllers.Redirect(http.StatusSeeOther, cms.ItemURL(item.Slug))
	}
	return renderEditForm(ctx, item, nil, nil, true, canPublish, "")
}

func (c *Controller) handleEdit(ctx *controllers.Context) controllers.Result {
	slug := routeContentSlug(ctx, "item_slug")
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
	canPublish, _ := cms.CanPublishItem(ctx.GoContext(), user.UserID, item.CategoryID)
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
	galleryJSON, embedsJSON, attachmentsJSON := cms.ItemMediaFromForm(ctx.Request)
	item.GalleryJSON = galleryJSON
	item.EmbedJSON = embedsJSON
	item.AttachmentsJSON = attachmentsJSON
	item.MetaTitle = strings.TrimSpace(ctx.Request.FormValue("meta_title"))
	item.MetaDescription = ctx.Request.FormValue("meta_description")
	item.MetaKeywords = strings.TrimSpace(ctx.Request.FormValue("meta_keywords"))
	item.CanonicalURL = strings.TrimSpace(ctx.Request.FormValue("canonical_url"))
	if canPublish {
		item.Featured = cms.FormBool(ctx.Request, "featured")
		item.PublishStart = cms.FormTimePtr(ctx.Request, "publish_start")
		item.PublishEnd = cms.FormTimePtr(ctx.Request, "publish_end")
		status := strings.TrimSpace(ctx.Request.FormValue("status"))
		if status == string(models.ItemStatusPublished) {
			item.Status = models.ItemStatusPublished
		} else {
			item.Status = models.ItemStatusDraft
		}
	} else if isNew {
		item.Status = models.ItemStatusDraft
	}

	var cat *models.Category
	if item.CategoryID != nil && *item.CategoryID > 0 {
		var c models.Category
		if db.First(&c, *item.CategoryID).Error == nil {
			cat = &c
		}
	}
	customFields, _ := cms.FieldsForCategory(ctx.GoContext(), cat)
	if err := cms.ValidateRequiredCustomFields(customFields, ctx.Request); err != nil {
		return err
	}

	if isNew {
		ok, err := cms.CanCreateItemInCategory(ctx.GoContext(), *item.AuthorID, item.CategoryID)
		if err != nil {
			return err
		}
		if !ok {
			return cms.ErrPermissionDenied
		}
	} else if item.AuthorID != nil {
		ok, err := cms.CanEditItem(ctx.GoContext(), *item.AuthorID, item)
		if err != nil {
			return err
		}
		if !ok {
			return cms.ErrPermissionDenied
		}
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
	if err := cms.SaveItemFieldValues(db, item.ItemID, customFields, ctx.Request); err != nil {
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
		"Gallery":            cms.ParseGalleryJSON(item.GalleryJSON),
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
