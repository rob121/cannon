package content

import (
	"context"
	"testing"

	"github.com/rob121/cannon/internal/config"
	"github.com/rob121/cannon/internal/database"
	"github.com/rob121/cannon/internal/groups"
	"github.com/rob121/cannon/internal/models"
	"github.com/rob121/cannon/internal/settings"
	"github.com/rob121/cannon/internal/sites"
	"gorm.io/gorm"
)

func TestSyncProfileAvatarFromSSO(t *testing.T) {
	ctx, db, profile, field := avatarProfileTestFixtures(t)
	user := models.User{Username: "avatar-user", Email: "avatar@example.com", Status: models.StatusActive, Validated: true}
	if err := db.Create(&user).Error; err != nil {
		t.Fatal(err)
	}
	if err := SyncProfileAvatarFromSSO(ctx, user.UserID, "https://cdn.example/avatar.jpg"); err != nil {
		t.Fatal(err)
	}
	var value models.ProfileUserFieldValue
	if err := db.Where("user_id = ? AND field_id = ?", user.UserID, field.ProfileFieldID).First(&value).Error; err != nil {
		t.Fatal(err)
	}
	if value.Value != "https://cdn.example/avatar.jpg" {
		t.Fatalf("value = %q", value.Value)
	}
	_ = profile
}

func avatarProfileTestFixtures(t *testing.T) (context.Context, *gorm.DB, models.Profile, models.ProfileField) {
	t.Helper()
	path := t.TempDir() + "/avatar.sqlite"
	site := &config.SiteConfig{
		ID: t.Name(),
		Database: config.DatabaseConfig{
			Type: "sqlite",
			DSN:  path,
		},
	}
	if err := database.Migrate(site); err != nil {
		t.Fatal(err)
	}
	db, err := database.Get(site.ID)
	if err != nil {
		t.Fatal(err)
	}
	if err := groups.EnsureDefaults(db); err != nil {
		t.Fatal(err)
	}
	profile := models.Profile{Name: "Author"}
	if err := db.Create(&profile).Error; err != nil {
		t.Fatal(err)
	}
	field := models.ProfileField{ProfileID: profile.ProfileID, Name: "Avatar", Type: "image", Status: models.StatusActive}
	if err := db.Create(&field).Error; err != nil {
		t.Fatal(err)
	}
	ctx := sites.WithContext(context.Background(), site)
	store := settings.NewStore()
	if err := store.Save(ctx, settings.ScopeGlobal, "content", map[string]any{
		"author_profile_id": profile.ProfileID,
	}); err != nil {
		t.Fatal(err)
	}
	return ctx, db, profile, field
}
