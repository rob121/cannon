package content

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/rob121/cannon/internal/hooks"
	"github.com/rob121/cannon/internal/models"
	"github.com/rob121/cannon/internal/sites"
	"gorm.io/gorm"
)

var (
	ErrCommentsDisabled = errors.New("comments are disabled")
	ErrLoginRequired    = errors.New("login required to comment")
	ErrAnonymousDenied  = errors.New("anonymous comments are not allowed")
)

// CommentInput is submitted comment data from a form.
type CommentInput struct {
	ItemID      uint
	UserID      *uint
	AuthorName  string
	AuthorEmail string
	Body        string
	IP          string
}

// CreateComment validates settings and saves a new comment.
func CreateComment(ctx context.Context, in CommentInput, authenticated bool) (*models.Comment, error) {
	cfg, err := LoadSettings(ctx)
	if err != nil {
		return nil, err
	}
	if !cfg.AllowComments {
		return nil, ErrCommentsDisabled
	}
	if cfg.RequireLogin && !authenticated {
		return nil, ErrLoginRequired
	}
	if !authenticated && !cfg.AllowAnonymous {
		return nil, ErrAnonymousDenied
	}
	body := strings.TrimSpace(in.Body)
	if body == "" {
		return nil, fmt.Errorf("comment body is required")
	}

	db, err := sites.DB(ctx)
	if err != nil {
		return nil, err
	}
	var item models.Item
	if err := db.First(&item, in.ItemID).Error; err != nil {
		return nil, err
	}

	row := models.Comment{
		ItemID:      in.ItemID,
		UserID:      in.UserID,
		AuthorName:  strings.TrimSpace(in.AuthorName),
		AuthorEmail: strings.TrimSpace(in.AuthorEmail),
		Body:        body,
		Approved:    cfg.AutoApprove,
		IP:          in.IP,
	}
	if authenticated && in.UserID != nil && row.AuthorName == "" {
		var user models.User
		if err := db.First(&user, *in.UserID).Error; err == nil {
			name := strings.TrimSpace(strings.TrimSpace(user.GivenName + " " + user.FamilyName))
			if name == "" {
				name = user.Username
			}
			row.AuthorName = name
			row.AuthorEmail = user.Email
		}
	}

	beforeArgs := map[string]any{"comment": &row, "item": &item}
	if _, err := hooks.Fire(ctx, hooks.OnCommentBeforeSave, beforeArgs); err != nil {
		return nil, err
	}
	if err := db.Create(&row).Error; err != nil {
		return nil, err
	}
	afterArgs := map[string]any{"comment_id": row.CommentID, "comment": &row, "item_id": item.ItemID}
	_, err = hooks.Fire(ctx, hooks.OnCommentAfterSave, afterArgs)
	return &row, err
}

// CommentCount returns approved comment count for an item.
func CommentCount(ctx context.Context, itemID uint) (int64, error) {
	db, err := sites.DB(ctx)
	if err != nil {
		return 0, err
	}
	var count int64
	err = db.Model(&models.Comment{}).Where("item_id = ? AND approved = ?", itemID, true).Count(&count).Error
	return count, err
}

// PendingCommentCount returns unapproved comments awaiting moderation.
func PendingCommentCount(ctx context.Context) (int64, error) {
	db, err := sites.DB(ctx)
	if err != nil {
		return 0, err
	}
	var count int64
	err = db.Model(&models.Comment{}).Where("approved = ?", false).Count(&count).Error
	return count, err
}

// RelatedItems returns published items related by category or shared tags.
func RelatedItems(ctx context.Context, viewerGroups []uint, item *models.Item, limit int) ([]models.Item, error) {
	if item == nil {
		return nil, nil
	}
	if limit <= 0 {
		limit = 5
	}
	q, err := VisibleItemsQuery(ctx, viewerGroups)
	if err != nil {
		return nil, err
	}
	q = q.Where("item_id <> ?", item.ItemID)
	tagIDs := make([]uint, 0, len(item.Tags))
	for _, t := range item.Tags {
		tagIDs = append(tagIDs, t.TagID)
	}
	switch {
	case len(tagIDs) > 0:
		q = q.Joins("JOIN item_tags ON item_tags.item_item_id = items.item_id AND item_tags.tag_tag_id IN ?", tagIDs)
	case item.CategoryID != nil && *item.CategoryID > 0:
		q = q.Where("category_id = ?", *item.CategoryID)
	default:
		return nil, nil
	}
	var rows []models.Item
	if err := q.Group("items.item_id").Order("items.created_at DESC").Limit(limit).Find(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}

// CategoryTemplate resolves the listing template for a category, walking parents when configured.
func CategoryTemplate(ctx context.Context, cat *models.Category) (string, error) {
	if cat == nil {
		return "", nil
	}
	db, err := sites.DB(ctx)
	if err != nil {
		return "", err
	}
	current := cat
	for {
		if strings.TrimSpace(current.Template) != "" {
			return current.Template, nil
		}
		if !current.InheritSettings || current.ParentID == nil || *current.ParentID == 0 {
			break
		}
		var parent models.Category
		if err := db.First(&parent, *current.ParentID).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				break
			}
			return "", err
		}
		current = &parent
	}
	return "", nil
}
