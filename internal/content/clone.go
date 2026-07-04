package content

import (
	"context"
	"strings"

	"github.com/rob121/cannon/internal/hooks"
	"github.com/rob121/cannon/internal/models"
	"gorm.io/gorm"
)

// CloneItem duplicates an item as a new draft, copying content, metadata, tags, groups, and custom fields.
func CloneItem(ctx context.Context, db *gorm.DB, sourceID uint) (*models.Item, error) {
	var src models.Item
	if err := db.Preload("Tags").Preload("Groups").First(&src, sourceID).Error; err != nil {
		return nil, err
	}

	clone := models.Item{
		Locale:           src.Locale,
		Title:            cloneItemTitle(src.Title),
		Intro:            src.Intro,
		Body:             src.Body,
		Status:           models.ItemStatusDraft,
		Featured:         false,
		FeaturedSort:     0,
		PublishStart:     src.PublishStart,
		PublishEnd:       src.PublishEnd,
		AuthorID:         src.AuthorID,
		CategoryID:       src.CategoryID,
		Image:            src.Image,
		GalleryJSON:      src.GalleryJSON,
		EmbedJSON:        src.EmbedJSON,
		AttachmentsJSON:  src.AttachmentsJSON,
		MetaTitle:        src.MetaTitle,
		MetaDescription:  src.MetaDescription,
		MetaKeywords:     src.MetaKeywords,
		Sort:             src.Sort,
	}
	NormalizeItemLocale(ctx, &clone)

	slugCtx := WithLocale(ctx, clone.Locale)
	slug, err := UniqueItemSlug(slugCtx, clone.Title, 0)
	if err != nil {
		return nil, err
	}
	clone.Slug = slug

	beforeArgs := map[string]any{
		"item":           &clone,
		"is_new":         true,
		"source_item_id": sourceID,
	}
	if _, err := hooks.Fire(ctx, hooks.OnItemBeforeSave, beforeArgs); err != nil {
		return nil, err
	}

	if err := db.Create(&clone).Error; err != nil {
		return nil, err
	}
	if len(src.Tags) > 0 {
		if err := db.Model(&clone).Association("Tags").Replace(src.Tags); err != nil {
			return nil, err
		}
	}
	if len(src.Groups) > 0 {
		if err := db.Model(&clone).Association("Groups").Replace(src.Groups); err != nil {
			return nil, err
		}
	}
	if err := copyItemFieldValues(db, sourceID, clone.ItemID); err != nil {
		return nil, err
	}
	if err := UpsertSearchIndex(ctx, db, &clone); err != nil {
		return nil, err
	}

	afterArgs := map[string]any{
		"item_id":        clone.ItemID,
		"item":           &clone,
		"source_item_id": sourceID,
	}
	_, err = hooks.Fire(ctx, hooks.OnItemAfterSave, afterArgs)
	if err != nil {
		return nil, err
	}
	return &clone, nil
}

func cloneItemTitle(title string) string {
	title = strings.TrimSpace(title)
	if title == "" {
		return "Copy of Item"
	}
	const prefix = "Copy of "
	if strings.HasPrefix(title, prefix) {
		return title
	}
	return prefix + title
}

func copyItemFieldValues(db *gorm.DB, sourceID, destID uint) error {
	var values []models.ItemFieldValue
	if err := db.Where("item_id = ?", sourceID).Find(&values).Error; err != nil {
		return err
	}
	for _, row := range values {
		if err := db.Create(&models.ItemFieldValue{
			ItemID:  destID,
			FieldID: row.FieldID,
			Value:   row.Value,
		}).Error; err != nil {
			return err
		}
	}
	return nil
}
