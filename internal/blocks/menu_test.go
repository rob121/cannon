package blocks

import (
	"context"
	"testing"

	"github.com/rob121/cannon/internal/config"
	"github.com/rob121/cannon/internal/database"
	"github.com/rob121/cannon/internal/groups"
	"github.com/rob121/cannon/internal/models"
	"github.com/rob121/cannon/internal/sites"
)

func TestRenderMenuVerticalBlock(t *testing.T) {
	ctx, menuName := menuBlockTestContext(t)
	rendered := ""
	render := func(name string, data map[string]any) (string, error) {
		if name != MenuVerticalBlockTemplate {
			t.Fatalf("template = %q", name)
		}
		items, _ := data["Items"].([]map[string]any)
		if len(items) != 1 || items[0]["Name"] != "Home" {
			t.Fatalf("items = %#v", data["Items"])
		}
		if data["Class"] != "sidebar-menu" {
			t.Fatalf("class = %#v", data["Class"])
		}
		rendered = "<nav>ok</nav>"
		return rendered, nil
	}
	got, err := RenderMenuVerticalBlock(ctx, Metadata{MenuName: menuName, MenuClass: "sidebar-menu"}, render)
	if err != nil {
		t.Fatal(err)
	}
	if got != rendered {
		t.Fatalf("got %q", got)
	}
}

func TestRenderMenuBlockEmptyMenuName(t *testing.T) {
	got, err := RenderMenuHorizontalBlock(context.Background(), Metadata{}, func(string, map[string]any) (string, error) {
		t.Fatal("render should not be called")
		return "", nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if got != "" {
		t.Fatalf("got %q", got)
	}
}

func menuBlockTestContext(t *testing.T) (context.Context, string) {
	t.Helper()
	path := t.TempDir() + "/menu-block.sqlite"
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
	menu := models.Menu{MenuName: "main", Status: models.StatusActive}
	if err := db.Create(&menu).Error; err != nil {
		t.Fatal(err)
	}
	item := models.MenuItem{MenuID: menu.MenuID, Name: "Home", Sort: 0}
	if err := db.Create(&item).Error; err != nil {
		t.Fatal(err)
	}
	return sites.WithContext(context.Background(), site), menu.MenuName
}
