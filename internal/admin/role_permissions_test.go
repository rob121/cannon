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

func TestRoleEffectivePermissionPreviewSkipsParentDeny(t *testing.T) {
	const stripeSetup = "cannon-ext-stripe-subscription.stripe.setup"

	site := &config.SiteConfig{
		ID: "role-perm-preview-test",
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

	var editor, administrator models.Role
	if err := db.Where("name = ?", security.RoleEditor).First(&editor).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Where("name = ?", security.RoleAdministrator).First(&administrator).Error; err != nil {
		t.Fatal(err)
	}
	if err := security.SaveRolePermissions(db, editor.RoleID, nil, []string{stripeSetup}); err != nil {
		t.Fatal(err)
	}
	security.InvalidateSite(site.ID)

	preview, err := roleEffectivePermissionPreview(ctx, administrator.RoleID)
	if err != nil {
		t.Fatal(err)
	}
	for _, rows := range preview.ByCategory {
		for _, row := range rows {
			if row.ID == stripeSetup && row.Denied {
				t.Fatal("administrator role preview should not include editor parent deny")
			}
		}
	}
}
