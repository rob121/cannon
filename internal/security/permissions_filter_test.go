package security

import (
	"testing"

	"github.com/rob121/cannon/internal/models"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestFilterPermissionsByInstalledExtensions(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&models.Extension{}); err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&models.Extension{Name: "cannon-ext-powwow", Installed: false, Status: models.StatusInactive}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&models.Extension{Name: "cannon-ext-stripe-subscription", Installed: true, Status: models.StatusActive}).Error; err != nil {
		t.Fatal(err)
	}

	perms := []Permission{
		{ID: "core.admin.access", Category: "Admin"},
		{ID: "cannon-ext-powwow.chat.read", Category: "cannon-ext-powwow"},
		{ID: "cannon-ext-stripe-subscription.stripe.setup", Category: "cannon-ext-stripe-subscription"},
	}
	filtered, err := FilterPermissionsByInstalledExtensions(db, perms)
	if err != nil {
		t.Fatal(err)
	}
	if len(filtered) != 2 {
		t.Fatalf("len = %d, want 2", len(filtered))
	}
	for _, p := range filtered {
		if p.ID == "cannon-ext-powwow.chat.read" {
			t.Fatal("uninstalled extension permission should be filtered")
		}
	}
}

func TestPermissionExtensionOwner(t *testing.T) {
	if ext, ok := PermissionExtensionOwner("cannon-ext-powwow.chat.read"); !ok || ext != "cannon-ext-powwow" {
		t.Fatalf("got %q ok=%v", ext, ok)
	}
	if _, ok := PermissionExtensionOwner("core.admin.users.read"); ok {
		t.Fatal("core permission should not have extension owner")
	}
}
