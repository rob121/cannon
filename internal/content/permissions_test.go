package content_test

import (
	"context"
	"testing"

	"github.com/rob121/cannon/internal/config"
	"github.com/rob121/cannon/internal/content"
	"github.com/rob121/cannon/internal/database"
	"github.com/rob121/cannon/internal/models"
	"github.com/rob121/cannon/internal/roles"
	"github.com/rob121/cannon/internal/security"
	"github.com/rob121/cannon/internal/sites"
	"gorm.io/gorm"
)

func TestCanCreateItemRequiresRolePermission(t *testing.T) {
	ctx, db := permissionsTestContext(t)

	allowed := models.User{Username: "creator", Email: "creator@test", Status: models.StatusActive}
	denied := models.User{Username: "guest", Email: "guest@test", Status: models.StatusActive}
	for _, u := range []*models.User{&allowed, &denied} {
		if err := db.Create(u).Error; err != nil {
			t.Fatal(err)
		}
	}
	grantWriterRole(t, db, allowed.UserID)

	ok, err := content.CanCreateItem(ctx, allowed.UserID)
	if err != nil || !ok {
		t.Fatalf("allowed user: ok=%v err=%v", ok, err)
	}
	ok, err = content.CanCreateItem(ctx, denied.UserID)
	if err != nil || ok {
		t.Fatalf("denied user: ok=%v err=%v", ok, err)
	}
}

func TestCanEditItemOwnVsAny(t *testing.T) {
	ctx, db := permissionsTestContext(t)

	writer := models.User{Username: "writer", Email: "writer@test", Status: models.StatusActive}
	editor := models.User{Username: "editor", Email: "editor@test", Status: models.StatusActive}
	for _, u := range []*models.User{&writer, &editor} {
		if err := db.Create(u).Error; err != nil {
			t.Fatal(err)
		}
	}
	grantWriterRole(t, db, writer.UserID)
	grantEditorRole(t, db, editor.UserID)

	ownItem := models.Item{Title: "Own", Slug: "own-item", AuthorID: &writer.UserID, Status: models.ItemStatusDraft}
	otherItem := models.Item{Title: "Other", Slug: "other-item", AuthorID: &editor.UserID, Status: models.ItemStatusDraft}
	for _, item := range []*models.Item{&ownItem, &otherItem} {
		if err := db.Create(item).Error; err != nil {
			t.Fatal(err)
		}
	}

	ok, err := content.CanEditItem(ctx, writer.UserID, &ownItem)
	if err != nil || !ok {
		t.Fatalf("writer own item: ok=%v err=%v", ok, err)
	}
	ok, err = content.CanEditItem(ctx, writer.UserID, &otherItem)
	if err != nil || ok {
		t.Fatalf("writer other item: ok=%v err=%v", ok, err)
	}
	ok, err = content.CanEditItem(ctx, editor.UserID, &otherItem)
	if err != nil || !ok {
		t.Fatalf("editor other item: ok=%v err=%v", ok, err)
	}
}

func TestCanCreateCommentRequiresPermission(t *testing.T) {
	ctx, db := permissionsTestContext(t)
	user := models.User{Username: "commenter", Email: "commenter@test", Status: models.StatusActive}
	if err := db.Create(&user).Error; err != nil {
		t.Fatal(err)
	}
	grantWriterRole(t, db, user.UserID)

	ok, err := content.CanCreateComment(ctx, user.UserID)
	if err != nil || !ok {
		t.Fatalf("writer should post comments: ok=%v err=%v", ok, err)
	}
}

func grantWriterRole(t *testing.T, db *gorm.DB, userID uint) {
	t.Helper()
	var role models.Role
	if err := db.Where("name = ?", security.RoleWriter).First(&role).Error; err != nil {
		t.Fatal(err)
	}
	var user models.User
	if err := db.First(&user, userID).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Model(&user).Association("Roles").Append(&role); err != nil {
		t.Fatal(err)
	}
}

func grantEditorRole(t *testing.T, db *gorm.DB, userID uint) {
	t.Helper()
	var role models.Role
	if err := db.Where("name = ?", security.RoleEditor).First(&role).Error; err != nil {
		t.Fatal(err)
	}
	var user models.User
	if err := db.First(&user, userID).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Model(&user).Association("Roles").Append(&role); err != nil {
		t.Fatal(err)
	}
}

func permissionsTestContext(t *testing.T) (context.Context, *gorm.DB) {
	t.Helper()
	path := t.TempDir() + "/permissions.sqlite"
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
	if err := roles.EnsureDefaults(db); err != nil {
		t.Fatal(err)
	}
	return sites.WithContext(context.Background(), site), db
}
