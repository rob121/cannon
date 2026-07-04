package content

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"time"

	"github.com/rob121/cannon/internal/models"
	"github.com/rob121/cannon/internal/routepath"
	"github.com/rob121/cannon/internal/sites"
	"gorm.io/gorm"
)

const previewTokenTTL = 7 * 24 * time.Hour

var ErrPreviewInvalid = errors.New("invalid or expired preview link")

func newPreviewTokenValue() (string, error) {
	buf := make([]byte, 24)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}

// EnsurePreviewToken returns a valid preview token for an item, creating or refreshing when needed.
func EnsurePreviewToken(ctx context.Context, itemID uint) (string, error) {
	db, err := sites.DB(ctx)
	if err != nil {
		return "", err
	}
	var item models.Item
	if err := db.First(&item, itemID).Error; err != nil {
		return "", err
	}
	now := time.Now()
	if item.PreviewToken != "" && item.PreviewExpiresAt != nil && item.PreviewExpiresAt.After(now) {
		return item.PreviewToken, nil
	}
	token, err := newPreviewTokenValue()
	if err != nil {
		return "", err
	}
	expires := now.Add(previewTokenTTL)
	if err := db.Model(&item).Updates(map[string]any{
		"preview_token":      token,
		"preview_expires_at": expires,
	}).Error; err != nil {
		return "", err
	}
	return token, nil
}

// ItemByPreviewToken loads an item by its secret preview token.
func ItemByPreviewToken(ctx context.Context, token string) (*models.Item, error) {
	db, err := sites.DB(ctx)
	if err != nil {
		return nil, err
	}
	var item models.Item
	if err := db.Where("preview_token = ?", token).First(&item).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrPreviewInvalid
		}
		return nil, err
	}
	if item.PreviewExpiresAt == nil || time.Now().After(*item.PreviewExpiresAt) {
		return nil, ErrPreviewInvalid
	}
	if item.Status == models.ItemStatusTrashed {
		return nil, ErrPreviewInvalid
	}
	return &item, nil
}

// PreviewURL builds the public preview URL for a token.
func PreviewURL(ctx context.Context, token string) string {
	return routepath.ControllerWithSuffix(ctx, "content", "preview", token)
}
