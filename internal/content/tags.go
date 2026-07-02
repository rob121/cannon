package content

import (
	"context"
	"time"

	"github.com/rob121/cannon/internal/models"
	"github.com/rob121/cannon/internal/sites"
)

// TagCount pairs a tag with its published item count.
type TagCount struct {
	Tag   models.Tag
	Count int64
}

// ListTags returns all tags ordered by name.
func ListTags(ctx context.Context) ([]models.Tag, error) {
	db, err := sites.DB(ctx)
	if err != nil {
		return nil, err
	}
	var rows []models.Tag
	if err := db.Order("name asc").Find(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}

// TagCloud returns tags with published item counts for display.
func TagCloud(ctx context.Context, viewerGroups []uint, limit int) ([]TagCount, error) {
	if limit <= 0 {
		limit = 30
	}
	db, err := sites.DB(ctx)
	if err != nil {
		return nil, err
	}
	now := time.Now()
	sub := PublishedScope(db.Model(&models.Item{}), now).
		Select("items.item_id").
		Where(`NOT EXISTS (
			SELECT 1 FROM item_groups ig WHERE ig.item_item_id = items.item_id
		) OR EXISTS (
			SELECT 1 FROM item_groups ig
			WHERE ig.item_item_id = items.item_id AND ig.group_group_id IN ?
		)`, viewerGroups)

	type row struct {
		TagID uint
		Count int64
	}
	var counts []row
	err = db.Table("tags").
		Select("tags.tag_id, COUNT(DISTINCT item_tags.item_item_id) AS count").
		Joins("JOIN item_tags ON item_tags.tag_tag_id = tags.tag_id").
		Joins("JOIN (?) AS visible ON visible.item_id = item_tags.item_item_id", sub).
		Group("tags.tag_id").
		Order("count DESC, tags.name ASC").
		Limit(limit).
		Scan(&counts).Error
	if err != nil {
		return nil, err
	}
	if len(counts) == 0 {
		return nil, nil
	}
	ids := make([]uint, len(counts))
	for i, c := range counts {
		ids[i] = c.TagID
	}
	var tags []models.Tag
	if err := db.Where("tag_id IN ?", ids).Find(&tags).Error; err != nil {
		return nil, err
	}
	byID := map[uint]models.Tag{}
	for _, t := range tags {
		byID[t.TagID] = t
	}
	out := make([]TagCount, 0, len(counts))
	for _, c := range counts {
		if tag, ok := byID[c.TagID]; ok {
			out = append(out, TagCount{Tag: tag, Count: c.Count})
		}
	}
	return out, nil
}

// PopularItems returns published items ordered by approved comment count.
func PopularItems(ctx context.Context, viewerGroups []uint, limit int) ([]models.Item, error) {
	if limit <= 0 {
		limit = 10
	}
	q, err := VisibleItemsQuery(ctx, viewerGroups)
	if err != nil {
		return nil, err
	}
	var rows []models.Item
	err = q.
		Select("items.*, COUNT(comments.comment_id) AS comment_count").
		Joins("LEFT JOIN comments ON comments.item_id = items.item_id AND comments.approved = ?", true).
		Group("items.item_id").
		Order("comment_count DESC, items.created_at DESC").
		Limit(limit).
		Find(&rows).Error
	return rows, err
}
