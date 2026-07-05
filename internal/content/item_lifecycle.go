package content

import (
	"context"

	"github.com/rob121/cannon/internal/hooks"
	"github.com/rob121/cannon/internal/models"
	"github.com/rob121/cannon/internal/sites"
	"gorm.io/gorm"
)

// TrashItem moves an item to the trash status and fires lifecycle hooks.
func TrashItem(ctx context.Context, db *gorm.DB, id uint) error {
	var item models.Item
	if err := db.First(&item, id).Error; err != nil {
		return err
	}
	if err := db.Model(&item).Update("status", models.ItemStatusTrashed).Error; err != nil {
		return err
	}
	_, _ = hooks.Fire(ctx, hooks.OnItemTrash, map[string]any{
		"item_id": id,
		"item":    item,
	})
	return nil
}

// RestoreItem restores a trashed item to draft and fires lifecycle hooks.
func RestoreItem(ctx context.Context, db *gorm.DB, id uint) error {
	var item models.Item
	if err := db.Where("item_id = ? AND status = ?", id, models.ItemStatusTrashed).First(&item).Error; err != nil {
		return err
	}
	if err := db.Model(&item).Update("status", models.ItemStatusDraft).Error; err != nil {
		return err
	}
	_, _ = hooks.Fire(ctx, hooks.OnItemRestore, map[string]any{
		"item_id": id,
		"item":    item,
	})
	return nil
}

// DeleteItemPermanent permanently deletes an item and fires lifecycle hooks.
func DeleteItemPermanent(ctx context.Context, db *gorm.DB, id uint) error {
	var item models.Item
	found := db.First(&item, id).Error == nil
	if found {
		if _, err := hooks.Fire(ctx, hooks.OnItemBeforeDelete, map[string]any{
			"item_id": id,
			"item":    item,
		}); err != nil {
			return err
		}
	}
	RemoveSearchIndex(db, id)
	db.Exec("DELETE FROM item_groups WHERE item_item_id = ?", id)
	db.Exec("DELETE FROM item_tags WHERE item_item_id = ?", id)
	db.Where("item_id = ?", id).Delete(&models.ItemFieldValue{})
	db.Where("item_id = ?", id).Delete(&models.ItemRevision{})
	db.Where("item_id = ?", id).Delete(&models.Comment{})
	if err := db.Delete(&models.Item{}, id).Error; err != nil {
		return err
	}
	if found {
		_, _ = hooks.Fire(ctx, hooks.OnItemAfterDelete, map[string]any{
			"item_id": id,
			"item":    item,
		})
	}
	return nil
}

// FireRevisionRestore dispatches onRevisionRestore before applying a rollback.
func FireRevisionRestore(ctx context.Context, itemID, revisionID uint) error {
	db, err := sites.DB(ctx)
	if err != nil {
		return err
	}
	var item models.Item
	if err := db.First(&item, itemID).Error; err != nil {
		return err
	}
	_, err = hooks.Fire(ctx, hooks.OnRevisionRestore, map[string]any{
		"item_id":     itemID,
		"revision_id": revisionID,
		"item":        item,
	})
	return err
}
