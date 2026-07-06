package admin

import (
	"testing"

	"github.com/rob121/cannon/internal/extensions"
	"github.com/rob121/cannon/internal/models"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestExtensionDisplayTitle(t *testing.T) {
	if got := extensionDisplayTitle("Contact Forms", "", "cannon-extension-contact"); got != "Contact Forms" {
		t.Fatalf("got %q", got)
	}
	if got := extensionDisplayTitle("", "Menu Label", "binary"); got != "Menu Label" {
		t.Fatalf("got %q", got)
	}
	if got := extensionDisplayTitle("", "", "binary"); got != "binary" {
		t.Fatalf("got %q", got)
	}
}

func TestMergeExtensionMetaUsesCachedRow(t *testing.T) {
	meta := mergeExtensionMeta(extensions.MetaSummary{Available: false}, models.Extension{
		Name:          "demo",
		Title:         "Demo",
		Description:   "Cached description",
		Version:       "1.0.0",
		UpdateURLBase: "https://github.com/rob121/demo/releases/download",
	})
	if !meta.Available || meta.Description != "Cached description" || meta.Title != "Demo" || meta.UpdateURLBase == "" {
		t.Fatalf("meta = %+v", meta)
	}
}

func TestExtensionReorder(t *testing.T) {
	db := testAdminDB(t)
	rows := []models.Extension{
		{ExtensionID: 1, Name: "a", Sort: 0, Socket: "/tmp/a.sock", Status: models.StatusInactive},
		{ExtensionID: 2, Name: "b", Sort: 1, Socket: "/tmp/b.sock", Status: models.StatusInactive},
		{ExtensionID: 3, Name: "c", Sort: 2, Socket: "/tmp/c.sock", Status: models.StatusInactive},
	}
	for _, row := range rows {
		if err := db.Create(&row).Error; err != nil {
			t.Fatal(err)
		}
	}
	if err := extensionReorder(db, 2, 1); err != nil {
		t.Fatal(err)
	}
	var ordered []models.Extension
	if err := db.Order("sort asc, extension_id asc").Find(&ordered).Error; err != nil {
		t.Fatal(err)
	}
	if len(ordered) != 3 || ordered[0].Name != "a" || ordered[1].Name != "c" || ordered[2].Name != "b" {
		t.Fatalf("ordered = %#v", ordered)
	}
}

func testAdminDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&models.Extension{}); err != nil {
		t.Fatal(err)
	}
	return db
}
