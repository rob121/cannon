package content

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/rob121/cannon/internal/models"
	"github.com/rob121/cannon/internal/sites"
	"github.com/rob121/cannon/internal/user"
	"gorm.io/gorm"
)

const settingAuthorProfileID = "author_profile_id"

// AuthorProfileField is one profile field value for author pages.
type AuthorProfileField struct {
	Name  string
	Label string
	Type  string
	Value string
	HTML  string
}

// AuthorProfile holds public author profile data.
type AuthorProfile struct {
	DisplayName string
	Email       string
	Fields      []AuthorProfileField
}

// LoadAuthorProfile loads profile field values for a user using the configured author profile schema.
func LoadAuthorProfile(ctx context.Context, userID uint) (*AuthorProfile, error) {
	if userID == 0 {
		return nil, nil
	}
	db, err := sites.DB(ctx)
	if err != nil {
		return nil, err
	}
	var u models.User
	if err := db.First(&u, userID).Error; err != nil {
		return nil, err
	}
	out := &AuthorProfile{
		DisplayName: user.DisplayName(&u),
		Email:       u.Email,
	}
	profileID, err := AuthorProfileID(ctx)
	if err != nil || profileID == 0 {
		return out, nil
	}
	fields, err := ActiveProfileFields(ctx, profileID)
	if err != nil || len(fields) == 0 {
		return out, err
	}
	values, err := ProfileUserFieldValues(db, userID, fields)
	if err != nil {
		return out, err
	}
	for _, f := range fields {
		value := values[f.ProfileFieldID]
		if value == "" {
			continue
		}
		out.Fields = append(out.Fields, AuthorProfileField{
			Name:  f.Name,
			Label: f.Name,
			Type:  f.Type,
			Value: value,
			HTML:  FormatFieldDisplayHTML(profileFieldAsContentField(f), value),
		})
	}
	return out, nil
}

// ActiveProfileFields returns active fields for a profile schema.
func ActiveProfileFields(ctx context.Context, profileID uint) ([]models.ProfileField, error) {
	if profileID == 0 {
		return nil, nil
	}
	db, err := sites.DB(ctx)
	if err != nil {
		return nil, err
	}
	var fields []models.ProfileField
	err = db.Where("profile_id = ? AND status = ?", profileID, models.StatusActive).
		Order("sort asc, profile_field_id asc").
		Find(&fields).Error
	return fields, err
}

// ProfileUserFieldValues returns stored values keyed by profile field id.
func ProfileUserFieldValues(db *gorm.DB, userID uint, fields []models.ProfileField) (map[uint]string, error) {
	out := map[uint]string{}
	if userID == 0 || len(fields) == 0 {
		return out, nil
	}
	fieldIDs := make([]uint, 0, len(fields))
	for _, f := range fields {
		fieldIDs = append(fieldIDs, f.ProfileFieldID)
	}
	var rows []models.ProfileUserFieldValue
	if err := db.Where("user_id = ? AND field_id IN ?", userID, fieldIDs).Find(&rows).Error; err != nil {
		return nil, err
	}
	for _, row := range rows {
		out[row.FieldID] = row.Value
	}
	return out, nil
}

// SaveProfileUserFieldValues stores profile field values from an admin user form.
func SaveProfileUserFieldValues(db *gorm.DB, userID uint, fields []models.ProfileField, r *http.Request) error {
	for _, field := range fields {
		value := ProfileFieldFormValue(field, r)
		var existing models.ProfileUserFieldValue
		err := db.Where("user_id = ? AND field_id = ?", userID, field.ProfileFieldID).First(&existing).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			if value == "" {
				continue
			}
			if err := db.Create(&models.ProfileUserFieldValue{
				UserID:  userID,
				FieldID: field.ProfileFieldID,
				Value:   value,
			}).Error; err != nil {
				return err
			}
			continue
		}
		if err != nil {
			return err
		}
		if value == "" {
			if err := db.Delete(&existing).Error; err != nil {
				return err
			}
			continue
		}
		existing.Value = value
		if err := db.Save(&existing).Error; err != nil {
			return err
		}
	}
	return nil
}

// ProfileFieldFormValue reads one profile field from a form submission.
func ProfileFieldFormValue(field models.ProfileField, r *http.Request) string {
	cf := profileFieldAsContentField(field)
	return CustomFieldFormValue(cf, r)
}

func profileFieldAsContentField(field models.ProfileField) models.ContentField {
	return models.ContentField{
		FieldID:       field.ProfileFieldID,
		Name:          field.Name,
		Label:         field.Name,
		Type:          field.Type,
		Configuration: field.Configuration,
	}
}

// ProfileFieldAsContentField adapts a profile field for shared field widgets and formatting.
func ProfileFieldAsContentField(field models.ProfileField) models.ContentField {
	return profileFieldAsContentField(field)
}

// ProfileFieldFormKey returns the form input name for a profile field.
func ProfileFieldFormKey(fieldID uint) string {
	return fmt.Sprintf("field_%d", fieldID)
}
