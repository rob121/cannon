package content_test

import (
	"context"
	"testing"

	"github.com/rob121/cannon/internal/config"
	"github.com/rob121/cannon/internal/content"
	"github.com/rob121/cannon/internal/database"
	"github.com/rob121/cannon/internal/groups"
	"github.com/rob121/cannon/internal/models"
	"github.com/rob121/cannon/internal/settings"
	"github.com/rob121/cannon/internal/sites"
	"gorm.io/gorm"
)

func TestLoadAuthorProfileUsesGlobalSetting(t *testing.T) {
	ctx, db := authorProfileTestContext(t)

	profile := models.Profile{Name: "Author"}
	if err := db.Create(&profile).Error; err != nil {
		t.Fatal(err)
	}
	field := models.ProfileField{ProfileID: profile.ProfileID, Name: "Bio", Type: "textarea", Status: models.StatusActive}
	if err := db.Create(&field).Error; err != nil {
		t.Fatal(err)
	}
	user := models.User{Username: "ada", Email: "ada@test", Status: models.StatusActive, GivenName: "Ada", FamilyName: "Lovelace"}
	if err := db.Create(&user).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&models.ProfileUserFieldValue{
		UserID:  user.UserID,
		FieldID: field.ProfileFieldID,
		Value:   "Mathematician and writer.",
	}).Error; err != nil {
		t.Fatal(err)
	}
	if err := settings.NewStore().Save(ctx, settings.ScopeGlobal, content.SettingsSection, map[string]any{
		"author_profile_id": profile.ProfileID,
	}); err != nil {
		t.Fatal(err)
	}

	got, err := content.LoadAuthorProfile(ctx, user.UserID)
	if err != nil {
		t.Fatal(err)
	}
	if got.DisplayName != "Ada Lovelace" {
		t.Fatalf("display name: got %q", got.DisplayName)
	}
	if len(got.Fields) != 1 || got.Fields[0].Value != "Mathematician and writer." {
		t.Fatalf("fields: %+v", got.Fields)
	}
}

func TestLoadAuthorProfileWithoutSettingReturnsBasicsOnly(t *testing.T) {
	ctx, db := authorProfileTestContext(t)
	user := models.User{Username: "plain", Email: "plain@test", Status: models.StatusActive}
	if err := db.Create(&user).Error; err != nil {
		t.Fatal(err)
	}
	got, err := content.LoadAuthorProfile(ctx, user.UserID)
	if err != nil {
		t.Fatal(err)
	}
	if got.DisplayName != "plain" {
		t.Fatalf("display name: got %q", got.DisplayName)
	}
	if len(got.Fields) != 0 {
		t.Fatalf("expected no fields without author profile setting, got %+v", got.Fields)
	}
}

func authorProfileTestContext(t *testing.T) (context.Context, *gorm.DB) {
	t.Helper()
	path := t.TempDir() + "/author-profile.sqlite"
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
	return sites.WithContext(context.Background(), site), db
}
