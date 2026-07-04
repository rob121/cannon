package content

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/rob121/cannon/internal/models"
	"github.com/rob121/cannon/internal/sites"
	"gorm.io/gorm"
)

const maxItemRevisions = 50

// ItemRevisionSnapshot is the persisted item state for rollback and compare.
type ItemRevisionSnapshot struct {
	Title           string             `json:"title"`
	Slug            string             `json:"slug"`
	Intro           string             `json:"intro"`
	Body            string             `json:"body"`
	Status          models.ItemStatus  `json:"status"`
	Featured        bool               `json:"featured"`
	FeaturedSort    int                `json:"featured_sort"`
	PublishStart    *time.Time         `json:"publish_start,omitempty"`
	PublishEnd      *time.Time         `json:"publish_end,omitempty"`
	CategoryID      *uint              `json:"category_id,omitempty"`
	AuthorID        *uint              `json:"author_id,omitempty"`
	Image           string             `json:"image"`
	GalleryJSON     string             `json:"gallery_json"`
	EmbedJSON       string             `json:"embed_json"`
	AttachmentsJSON string             `json:"attachments_json"`
	MetaTitle       string             `json:"meta_title"`
	MetaDescription string             `json:"meta_description"`
	MetaKeywords    string             `json:"meta_keywords"`
	CanonicalURL    string             `json:"canonical_url"`
	Sort            int                `json:"sort"`
	TagIDs          []uint             `json:"tag_ids,omitempty"`
	GroupIDs        []uint             `json:"group_ids,omitempty"`
	FieldValues     map[uint]string    `json:"field_values,omitempty"`
}

// RevisionDiff describes one changed field between two snapshots.
type RevisionDiff struct {
	Field    string
	Label    string
	Previous string
	Current  string
}

// SnapshotItem builds a revision snapshot from a loaded item row.
func SnapshotItem(item *models.Item, tagIDs, groupIDs []uint, fieldValues map[uint]string) ItemRevisionSnapshot {
	return ItemRevisionSnapshot{
		Title:           item.Title,
		Slug:            item.Slug,
		Intro:           item.Intro,
		Body:            item.Body,
		Status:          item.Status,
		Featured:        item.Featured,
		FeaturedSort:    item.FeaturedSort,
		PublishStart:    item.PublishStart,
		PublishEnd:      item.PublishEnd,
		CategoryID:      item.CategoryID,
		AuthorID:        item.AuthorID,
		Image:           item.Image,
		GalleryJSON:     item.GalleryJSON,
		EmbedJSON:       item.EmbedJSON,
		AttachmentsJSON: item.AttachmentsJSON,
		MetaTitle:       item.MetaTitle,
		MetaDescription: item.MetaDescription,
		MetaKeywords:    item.MetaKeywords,
		CanonicalURL:    item.CanonicalURL,
		Sort:            item.Sort,
		TagIDs:          tagIDs,
		GroupIDs:        groupIDs,
		FieldValues:     fieldValues,
	}
}

// CreateItemRevision stores a snapshot of the current item before an update.
func CreateItemRevision(ctx context.Context, db *gorm.DB, itemID uint, editorID *uint, editorName string) error {
	var item models.Item
	if err := db.Preload("Tags").Preload("Groups").First(&item, itemID).Error; err != nil {
		return err
	}
	tagIDs := make([]uint, 0, len(item.Tags))
	for _, t := range item.Tags {
		tagIDs = append(tagIDs, t.TagID)
	}
	groupIDs := make([]uint, 0, len(item.Groups))
	for _, g := range item.Groups {
		groupIDs = append(groupIDs, g.GroupID)
	}
	fieldValues := map[uint]string{}
	var values []models.ItemFieldValue
	if err := db.Where("item_id = ?", itemID).Find(&values).Error; err != nil {
		return err
	}
	for _, v := range values {
		fieldValues[v.FieldID] = v.Value
	}
	snapshot := SnapshotItem(&item, tagIDs, groupIDs, fieldValues)
	raw, err := json.Marshal(snapshot)
	if err != nil {
		return err
	}
	var last models.ItemRevision
	next := 1
	if err := db.Where("item_id = ?", itemID).Order("revision_number desc").First(&last).Error; err == nil {
		next = last.RevisionNumber + 1
	}
	row := models.ItemRevision{
		ItemID:         itemID,
		RevisionNumber: next,
		SnapshotJSON:   string(raw),
		EditorID:       editorID,
		EditorName:     editorName,
	}
	if err := db.Create(&row).Error; err != nil {
		return err
	}
	return pruneItemRevisions(db, itemID)
}

func pruneItemRevisions(db *gorm.DB, itemID uint) error {
	var ids []uint
	if err := db.Model(&models.ItemRevision{}).
		Where("item_id = ?", itemID).
		Order("revision_number desc").
		Offset(maxItemRevisions).
		Pluck("revision_id", &ids).Error; err != nil {
		return err
	}
	if len(ids) == 0 {
		return nil
	}
	return db.Where("revision_id IN ?", ids).Delete(&models.ItemRevision{}).Error
}

// ListItemRevisions returns revisions newest first.
func ListItemRevisions(ctx context.Context, itemID uint) ([]models.ItemRevision, error) {
	db, err := sites.DB(ctx)
	if err != nil {
		return nil, err
	}
	var rows []models.ItemRevision
	if err := db.Where("item_id = ?", itemID).Order("revision_number desc").Find(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}

// LoadRevisionSnapshot decodes a revision snapshot.
func LoadRevisionSnapshot(rev models.ItemRevision) (ItemRevisionSnapshot, error) {
	var snap ItemRevisionSnapshot
	if err := json.Unmarshal([]byte(rev.SnapshotJSON), &snap); err != nil {
		return ItemRevisionSnapshot{}, err
	}
	return snap, nil
}

// CompareRevisionSnapshots returns human-readable field diffs.
func CompareRevisionSnapshots(a, b ItemRevisionSnapshot) []RevisionDiff {
	checks := []struct {
		field string
		label string
		left  string
		right string
	}{
		{"title", "Title", a.Title, b.Title},
		{"slug", "Slug", a.Slug, b.Slug},
		{"intro", "Intro", a.Intro, b.Intro},
		{"body", "Body", a.Body, b.Body},
		{"status", "Status", string(a.Status), string(b.Status)},
		{"meta_title", "Meta Title", a.MetaTitle, b.MetaTitle},
		{"meta_description", "Meta Description", a.MetaDescription, b.MetaDescription},
		{"meta_keywords", "Meta Keywords", a.MetaKeywords, b.MetaKeywords},
		{"image", "Featured Image", a.Image, b.Image},
	}
	var diffs []RevisionDiff
	for _, c := range checks {
		if c.left == c.right {
			continue
		}
		diffs = append(diffs, RevisionDiff{
			Field:    c.field,
			Label:    c.label,
			Previous: c.left,
			Current:  c.right,
		})
	}
	return diffs
}

// RollbackItemRevision restores an item from a stored revision.
func RollbackItemRevision(ctx context.Context, itemID, revisionID uint, editorID *uint, editorName string) error {
	db, err := sites.DB(ctx)
	if err != nil {
		return err
	}
	var rev models.ItemRevision
	if err := db.Where("item_id = ? AND revision_id = ?", itemID, revisionID).First(&rev).Error; err != nil {
		return err
	}
	snap, err := LoadRevisionSnapshot(rev)
	if err != nil {
		return err
	}
	if err := CreateItemRevision(ctx, db, itemID, editorID, editorName); err != nil {
		return err
	}
	var item models.Item
	if err := db.First(&item, itemID).Error; err != nil {
		return err
	}
	item.Title = snap.Title
	item.Slug = snap.Slug
	item.Intro = snap.Intro
	item.Body = snap.Body
	item.Status = snap.Status
	item.Featured = snap.Featured
	item.FeaturedSort = snap.FeaturedSort
	item.PublishStart = snap.PublishStart
	item.PublishEnd = snap.PublishEnd
	item.CategoryID = snap.CategoryID
	item.AuthorID = snap.AuthorID
	item.Image = snap.Image
	item.GalleryJSON = snap.GalleryJSON
	item.EmbedJSON = snap.EmbedJSON
	item.AttachmentsJSON = snap.AttachmentsJSON
	item.MetaTitle = snap.MetaTitle
	item.MetaDescription = snap.MetaDescription
	item.MetaKeywords = snap.MetaKeywords
	item.CanonicalURL = snap.CanonicalURL
	item.Sort = snap.Sort
	if err := db.Save(&item).Error; err != nil {
		return err
	}
	tags := make([]models.Tag, 0, len(snap.TagIDs))
	for _, id := range snap.TagIDs {
		tags = append(tags, models.Tag{TagID: id})
	}
	if err := db.Model(&item).Association("Tags").Replace(tags); err != nil {
		return err
	}
	groups := make([]models.Group, 0, len(snap.GroupIDs))
	for _, id := range snap.GroupIDs {
		groups = append(groups, models.Group{GroupID: id})
	}
	if err := db.Model(&item).Association("Groups").Replace(groups); err != nil {
		return err
	}
	if err := db.Where("item_id = ?", itemID).Delete(&models.ItemFieldValue{}).Error; err != nil {
		return err
	}
	for fieldID, value := range snap.FieldValues {
		if err := db.Create(&models.ItemFieldValue{ItemID: itemID, FieldID: fieldID, Value: value}).Error; err != nil {
			return err
		}
	}
	return nil
}

// PendingItemCount returns items awaiting review.
func PendingItemCount(ctx context.Context) (int64, error) {
	db, err := sites.DB(ctx)
	if err != nil {
		return 0, err
	}
	var count int64
	if err := db.Model(&models.Item{}).Where("status = ?", models.ItemStatusPending).Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

// TrashedItemCount returns items in trash.
func TrashedItemCount(ctx context.Context) (int64, error) {
	db, err := sites.DB(ctx)
	if err != nil {
		return 0, err
	}
	var count int64
	if err := db.Model(&models.Item{}).Where("status = ?", models.ItemStatusTrashed).Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

// RevisionLabel returns a display label for a revision row.
func RevisionLabel(rev models.ItemRevision) string {
	return fmt.Sprintf("Revision %d", rev.RevisionNumber)
}
