package admin

import (
	"context"
	"testing"

	"github.com/rob121/cannon/internal/config"
	"github.com/rob121/cannon/internal/database"
	"github.com/rob121/cannon/internal/models"
	"github.com/rob121/cannon/internal/roles"
	"github.com/rob121/cannon/internal/security"
	"github.com/rob121/cannon/internal/sites"
)

func TestEffectivePermissionPreview(t *testing.T) {
	site := &config.SiteConfig{
		ID: "user-perm-preview-test",
		Database: config.DatabaseConfig{
			Type: "sqlite",
			DSN:  t.TempDir() + "/site.db",
		},
	}
	if err := database.Migrate(site); err != nil {
		t.Fatal(err)
	}
	db, err := database.Get(site.ID)
	if err != nil {
		t.Fatal(err)
	}
	if err := roles.EnsureDefaults(db); err != nil {
		t.Fatal(err)
	}
	ctx := sites.WithContext(context.Background(), site)

	var writer models.Role
	if err := db.Where("name = ?", security.RoleWriter).First(&writer).Error; err != nil {
		t.Fatal(err)
	}
	user := models.User{Username: "writer", Email: "writer@test", Status: models.StatusActive}
	if err := db.Create(&user).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Model(&user).Association("Roles").Append(&writer); err != nil {
		t.Fatal(err)
	}

	preview, err := effectivePermissionPreview(ctx, user.UserID)
	if err != nil {
		t.Fatal(err)
	}
	if preview.Total == 0 {
		t.Fatal("expected effective permissions for writer role")
	}
	found := false
	for _, rows := range preview.ByCategory {
		for _, row := range rows {
			if row.ID == security.PermContentFrontendCreate {
				found = true
				break
			}
		}
	}
	if !found {
		t.Fatalf("expected writer create permission in preview: %#v", preview)
	}
}
