package content_test

import (
	"context"
	"testing"

	"github.com/rob121/cannon/internal/config"
	"github.com/rob121/cannon/internal/content"
	"github.com/rob121/cannon/internal/database"
	"github.com/rob121/cannon/internal/models"
	"github.com/rob121/cannon/internal/sites"
)

func TestNormalizeCategorySlug(t *testing.T) {
	if got := content.NormalizeCategorySlug(" News / Local News "); got != "news/local-news" {
		t.Fatalf("got %q", got)
	}
}

func TestResolveCategorySlugUsesParentPath(t *testing.T) {
	ctx, parentID := categoryTestContext(t)
	parent := uint(parentID)

	slug, err := content.ResolveCategorySlug(ctx, "Local", "", &parent, 0)
	if err != nil {
		t.Fatal(err)
	}
	if slug != "news/local" {
		t.Fatalf("slug = %q, want news/local", slug)
	}
}

func TestResolveCategorySlugAvoidsParentCollision(t *testing.T) {
	ctx, parentID := categoryTestContext(t)
	parent := uint(parentID)

	slug, err := content.ResolveCategorySlug(ctx, "News", "", &parent, 0)
	if err != nil {
		t.Fatal(err)
	}
	if slug != "news/news" {
		t.Fatalf("slug = %q, want news/news", slug)
	}
}

func TestResolveCategorySlugUniquifiesManualSlug(t *testing.T) {
	ctx, parentID := categoryTestContext(t)
	parent := uint(parentID)

	slug, err := content.ResolveCategorySlug(ctx, "Other", "news", &parent, 0)
	if err != nil {
		t.Fatal(err)
	}
	if slug != "news-1" {
		t.Fatalf("slug = %q, want news-1", slug)
	}
}

func categoryTestContext(t *testing.T) (context.Context, uint) {
	t.Helper()
	path := t.TempDir() + "/categories.sqlite"
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
	parent := models.Category{Name: "News", Slug: "news", Status: models.StatusActive}
	if err := db.Create(&parent).Error; err != nil {
		t.Fatal(err)
	}
	return sites.WithContext(context.Background(), site), parent.CategoryID
}
