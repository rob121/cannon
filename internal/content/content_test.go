package content_test

import (
	"context"
	"testing"

	"github.com/rob121/cannon/internal/config"
	"github.com/rob121/cannon/internal/content"
	"github.com/rob121/cannon/internal/database"
	"github.com/rob121/cannon/internal/groups"
	"github.com/rob121/cannon/internal/models"
	"github.com/rob121/cannon/internal/sites"
)

func TestSlugify(t *testing.T) {
	if got := content.Slugify("Hello World!"); got != "hello-world" {
		t.Fatalf("got %q", got)
	}
}

func TestCategoryBySlugAllowsUnrestrictedCategory(t *testing.T) {
	ctx, publicID := categoryBySlugTestContext(t)

	cat, err := content.CategoryBySlug(ctx, "home", []uint{publicID})
	if err != nil {
		t.Fatalf("CategoryBySlug() error = %v", err)
	}
	if cat.Slug != "home" {
		t.Fatalf("slug = %q, want home", cat.Slug)
	}
}

func categoryBySlugTestContext(t *testing.T) (context.Context, uint) {
	t.Helper()
	path := t.TempDir() + "/category-by-slug.sqlite"
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
	publicID, err := groups.PublicGroupID(db)
	if err != nil {
		t.Fatal(err)
	}
	row := models.Category{Name: "Home", Slug: "home", Locale: "en-US", Status: models.StatusActive}
	if err := db.Create(&row).Error; err != nil {
		t.Fatal(err)
	}
	return sites.WithContext(context.Background(), site), publicID
}
