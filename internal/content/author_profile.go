package content

import (
	"context"

	"github.com/rob121/cannon/internal/models"
	"github.com/rob121/cannon/internal/sites"
	"github.com/rob121/cannon/internal/user"
)

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

// LoadAuthorProfile loads profile field values for a user.
func LoadAuthorProfile(ctx context.Context, userID uint) (*AuthorProfile, error) {
	if userID == 0 {
		return nil, nil
	}
	var u models.User
	db, err := sites.DB(ctx)
	if err != nil {
		return nil, err
	}
	if err := db.First(&u, userID).Error; err != nil {
		return nil, err
	}
	out := &AuthorProfile{
		DisplayName: user.DisplayName(&u),
		Email:       u.Email,
	}
	var up models.UserProfile
	if err := db.Where("user_id = ?", userID).First(&up).Error; err != nil {
		return out, nil
	}
	var profile models.Profile
	if err := db.Preload("Fields", "status = ?", models.StatusActive).First(&profile, up.ProfileID).Error; err != nil {
		return out, nil
	}
	if len(profile.Fields) == 0 {
		return out, nil
	}
	fieldIDs := make([]uint, 0, len(profile.Fields))
	for _, f := range profile.Fields {
		fieldIDs = append(fieldIDs, f.ProfileFieldID)
	}
	var values []models.ProfileUserFieldValue
	_ = db.Where("user_id = ? AND field_id IN ?", userID, fieldIDs).Find(&values).Error
	byField := make(map[uint]string, len(values))
	for _, v := range values {
		byField[v.FieldID] = v.Value
	}
	for _, f := range profile.Fields {
		value := byField[f.ProfileFieldID]
		if value == "" {
			continue
		}
		label := f.Name
		cf := models.ContentField{Label: f.Name, Type: f.Type, Configuration: f.Configuration}
		out.Fields = append(out.Fields, AuthorProfileField{
			Name:  f.Name,
			Label: label,
			Type:  f.Type,
			Value: value,
			HTML:  FormatFieldDisplayHTML(cf, value),
		})
	}
	return out, nil
}
