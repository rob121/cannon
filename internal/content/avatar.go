package content

import (
	"context"
	"errors"
	"strings"

	"github.com/rob121/cannon/internal/models"
	"github.com/rob121/cannon/internal/sites"
	"github.com/rob121/cannon/internal/user"
	"gorm.io/gorm"
)

// ResolveUserAvatar returns the best avatar URL for a user, including profile image fields.
func ResolveUserAvatar(ctx context.Context, userID uint) (string, error) {
	if userID == 0 {
		return "", nil
	}
	db, err := sites.DB(ctx)
	if err != nil {
		return "", err
	}
	var u models.User
	if err := db.First(&u, userID).Error; err != nil {
		return "", err
	}
	if v := user.ResolveAvatar(&u); v != "" {
		return v, nil
	}
	profileID, err := AuthorProfileID(ctx)
	if err != nil || profileID == 0 {
		return "", err
	}
	fields, err := ActiveProfileFields(ctx, profileID)
	if err != nil {
		return "", err
	}
	field, ok := avatarProfileField(fields)
	if !ok {
		return "", nil
	}
	values, err := ProfileUserFieldValues(db, userID, []models.ProfileField{field})
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(values[field.ProfileFieldID]), nil
}

// SyncProfileAvatarFromSSO stores a provider avatar in the author profile when no custom avatar exists.
func SyncProfileAvatarFromSSO(ctx context.Context, userID uint, avatarURL string) error {
	avatarURL = strings.TrimSpace(avatarURL)
	if userID == 0 || avatarURL == "" {
		return nil
	}
	db, err := sites.DB(ctx)
	if err != nil {
		return err
	}
	var u models.User
	if err := db.First(&u, userID).Error; err != nil {
		return err
	}
	if strings.TrimSpace(u.AvatarURL) != "" {
		return nil
	}
	profileID, err := AuthorProfileID(ctx)
	if err != nil || profileID == 0 {
		return nil
	}
	fields, err := ActiveProfileFields(ctx, profileID)
	if err != nil {
		return err
	}
	field, ok := avatarProfileField(fields)
	if !ok {
		return nil
	}
	var existing models.ProfileUserFieldValue
	err = db.Where("user_id = ? AND field_id = ?", userID, field.ProfileFieldID).First(&existing).Error
	if err == nil && strings.TrimSpace(existing.Value) != "" {
		return nil
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return db.Create(&models.ProfileUserFieldValue{
			UserID:  userID,
			FieldID: field.ProfileFieldID,
			Value:   avatarURL,
		}).Error
	}
	return err
}

func avatarProfileField(fields []models.ProfileField) (models.ProfileField, bool) {
	for _, field := range fields {
		if field.Type != "image" {
			continue
		}
		name := strings.ToLower(strings.TrimSpace(field.Name))
		if name == "avatar" || name == "photo" || name == "profile photo" || name == "profile image" {
			return field, true
		}
	}
	return models.ProfileField{}, false
}
