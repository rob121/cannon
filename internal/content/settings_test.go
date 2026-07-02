package content

import (
	"context"
	"testing"

	"github.com/rob121/cannon/internal/config"
	"github.com/rob121/cannon/internal/database"
	"github.com/rob121/cannon/internal/models"
	"github.com/rob121/cannon/internal/settings"
	"github.com/rob121/cannon/internal/sites"
)

func TestCreateCommentRespectsShowComments(t *testing.T) {
	ctx := commentSettingsContext(t, map[string]any{
		"show_comments":   false,
		"allow_comments":  true,
		"allow_anonymous": true,
		"require_login":   false,
	})
	_, err := CreateComment(ctx, CommentInput{ItemID: 1, Body: "hello"}, false)
	if err != ErrCommentsDisabled {
		t.Fatalf("expected ErrCommentsDisabled, got %v", err)
	}
}

func TestLoadSettingsShowCommentsDefault(t *testing.T) {
	ctx := commentSettingsContext(t, map[string]any{
		"allow_comments": true,
	})
	cfg, err := LoadSettings(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if !cfg.ShowComments {
		t.Fatal("show_comments should default true")
	}
}

func commentSettingsContext(t *testing.T, data map[string]any) context.Context {
	t.Helper()
	path := t.TempDir() + "/content-settings.sqlite"
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
	ctx := sites.WithContext(context.Background(), site)
	if err := settings.NewStore().Save(ctx, settings.ScopeGlobal, SettingsSection, data); err != nil {
		t.Fatal(err)
	}
	db, err := sites.DB(ctx)
	if err != nil {
		t.Fatal(err)
	}
	item := models.Item{ItemID: 1, Title: "Test", Slug: "test", Status: models.ItemStatusPublished}
	if err := db.Create(&item).Error; err != nil {
		t.Fatal(err)
	}
	return ctx
}
