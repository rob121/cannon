package content_test

import (
	"context"
	"testing"

	"github.com/rob121/cannon/internal/config"
	"github.com/rob121/cannon/internal/content"
	"github.com/rob121/cannon/internal/database"
	"github.com/rob121/cannon/internal/groups"
	"github.com/rob121/cannon/internal/models"
	"github.com/rob121/cannon/internal/roles"
	"github.com/rob121/cannon/internal/settings"
	"github.com/rob121/cannon/internal/sites"
	"gorm.io/gorm"
)

func TestCanCreateItemUsesGlobalGroups(t *testing.T) {
	ctx, db := permissionsTestContext(t)

	createGroup := models.Group{Name: "Creators", Kind: models.GroupKindFrontend, Status: models.StatusActive}
	if err := db.Create(&createGroup).Error; err != nil {
		t.Fatal(err)
	}
	store := settings.NewStore()
	if err := store.Save(ctx, settings.ScopeGlobal, content.SettingsSection, map[string]any{
		"create_group_ids": []any{createGroup.GroupID},
	}); err != nil {
		t.Fatal(err)
	}

	allowed := models.User{Username: "creator", Email: "creator@test", Status: models.StatusActive}
	denied := models.User{Username: "guest", Email: "guest@test", Status: models.StatusActive}
	for _, u := range []*models.User{&allowed, &denied} {
		if err := db.Create(u).Error; err != nil {
			t.Fatal(err)
		}
	}
	if err := db.Model(&allowed).Association("Groups").Replace([]models.Group{createGroup}); err != nil {
		t.Fatal(err)
	}

	ok, err := content.CanCreateItem(ctx, allowed.UserID)
	if err != nil || !ok {
		t.Fatalf("allowed user: ok=%v err=%v", ok, err)
	}
	ok, err = content.CanCreateItem(ctx, denied.UserID)
	if err != nil || ok {
		t.Fatalf("denied user: ok=%v err=%v", ok, err)
	}
}

func TestCategoryInheritsGlobalCreateGroups(t *testing.T) {
	ctx, db := permissionsTestContext(t)

	createGroup := models.Group{Name: "Global Creators", Kind: models.GroupKindFrontend, Status: models.StatusActive}
	if err := db.Create(&createGroup).Error; err != nil {
		t.Fatal(err)
	}
	store := settings.NewStore()
	if err := store.Save(ctx, settings.ScopeGlobal, content.SettingsSection, map[string]any{
		"create_group_ids": []any{createGroup.GroupID},
	}); err != nil {
		t.Fatal(err)
	}
	cat := models.Category{Name: "News", Slug: "news", Status: models.StatusActive, InheritPermissions: true}
	if err := db.Create(&cat).Error; err != nil {
		t.Fatal(err)
	}

	user := models.User{Username: "creator2", Email: "creator2@test", Status: models.StatusActive}
	if err := db.Create(&user).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Model(&user).Association("Groups").Replace([]models.Group{createGroup}); err != nil {
		t.Fatal(err)
	}

	catID := cat.CategoryID
	ok, err := content.CanCreateItemInCategory(ctx, user.UserID, &catID)
	if err != nil || !ok {
		t.Fatalf("expected global create permission in category: ok=%v err=%v", ok, err)
	}
}

func TestCategoryOverridesGlobalCreateGroups(t *testing.T) {
	ctx, db := permissionsTestContext(t)

	globalGroup := models.Group{Name: "Global", Kind: models.GroupKindFrontend, Status: models.StatusActive}
	localGroup := models.Group{Name: "Local Only", Kind: models.GroupKindFrontend, Status: models.StatusActive}
	for _, g := range []*models.Group{&globalGroup, &localGroup} {
		if err := db.Create(g).Error; err != nil {
			t.Fatal(err)
		}
	}
	store := settings.NewStore()
	if err := store.Save(ctx, settings.ScopeGlobal, content.SettingsSection, map[string]any{
		"create_group_ids": []any{globalGroup.GroupID},
	}); err != nil {
		t.Fatal(err)
	}
	cat := models.Category{Name: "Restricted", Slug: "restricted-local", Status: models.StatusActive, InheritPermissions: false}
	if err := db.Create(&cat).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Model(&cat).Association("CreateGroups").Replace([]models.Group{localGroup}); err != nil {
		t.Fatal(err)
	}

	globalUser := models.User{Username: "global-user", Email: "global-user@test", Status: models.StatusActive}
	localUser := models.User{Username: "local-user", Email: "local-user@test", Status: models.StatusActive}
	for _, u := range []*models.User{&globalUser, &localUser} {
		if err := db.Create(u).Error; err != nil {
			t.Fatal(err)
		}
	}
	if err := db.Model(&globalUser).Association("Groups").Replace([]models.Group{globalGroup}); err != nil {
		t.Fatal(err)
	}
	if err := db.Model(&localUser).Association("Groups").Replace([]models.Group{localGroup}); err != nil {
		t.Fatal(err)
	}

	catID := cat.CategoryID
	ok, err := content.CanCreateItemInCategory(ctx, localUser.UserID, &catID)
	if err != nil || !ok {
		t.Fatalf("local user: ok=%v err=%v", ok, err)
	}
	ok, err = content.CanCreateItemInCategory(ctx, globalUser.UserID, &catID)
	if err != nil || ok {
		t.Fatalf("global-only user should be denied: ok=%v err=%v", ok, err)
	}
}

func TestCanCreateItemInCategoryRequiresGroup(t *testing.T) {
	ctx, db := permissionsTestContext(t)

	writerGroup := models.Group{Name: "Writers Only", Kind: models.GroupKindFrontend, Status: models.StatusActive}
	if err := db.Create(&writerGroup).Error; err != nil {
		t.Fatal(err)
	}
	cat := models.Category{Name: "Restricted", Slug: "restricted", Status: models.StatusActive, InheritPermissions: false}
	if err := db.Create(&cat).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Model(&cat).Association("CreateGroups").Replace([]models.Group{writerGroup}); err != nil {
		t.Fatal(err)
	}

	writerRole := models.Role{Name: roles.WriterRole, Status: models.StatusActive}
	if err := db.Create(&writerRole).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Model(&writerGroup).Association("Roles").Replace([]models.Role{writerRole}); err != nil {
		t.Fatal(err)
	}

	allowedUser := models.User{Username: "writer1", Email: "writer1@test", Status: models.StatusActive}
	deniedUser := models.User{Username: "writer2", Email: "writer2@test", Status: models.StatusActive}
	for _, u := range []*models.User{&allowedUser, &deniedUser} {
		if err := db.Create(u).Error; err != nil {
			t.Fatal(err)
		}
	}
	if err := db.Model(&allowedUser).Association("Groups").Replace([]models.Group{writerGroup}); err != nil {
		t.Fatal(err)
	}

	catID := cat.CategoryID
	ok, err := content.CanCreateItemInCategory(ctx, allowedUser.UserID, &catID)
	if err != nil || !ok {
		t.Fatalf("allowed user: ok=%v err=%v", ok, err)
	}
	ok, err = content.CanCreateItemInCategory(ctx, deniedUser.UserID, &catID)
	if err != nil || ok {
		t.Fatalf("denied user: ok=%v err=%v", ok, err)
	}
}

func TestCanEditItemRespectsCategoryEditGroups(t *testing.T) {
	ctx, db := permissionsTestContext(t)

	editGroup := models.Group{Name: "Editors Only", Kind: models.GroupKindFrontend, Status: models.StatusActive}
	if err := db.Create(&editGroup).Error; err != nil {
		t.Fatal(err)
	}
	cat := models.Category{Name: "Edit Restricted", Slug: "edit-restricted", Status: models.StatusActive, InheritPermissions: false}
	if err := db.Create(&cat).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Model(&cat).Association("EditGroups").Replace([]models.Group{editGroup}); err != nil {
		t.Fatal(err)
	}

	editorRole := models.Role{Name: roles.EditorRole, Status: models.StatusActive}
	if err := db.Create(&editorRole).Error; err != nil {
		t.Fatal(err)
	}
	adminGroup := models.Group{Name: "Admins", Kind: models.GroupKindBackend, Status: models.StatusActive}
	if err := db.Create(&adminGroup).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Model(&adminGroup).Association("Roles").Replace([]models.Role{editorRole}); err != nil {
		t.Fatal(err)
	}

	allowed := models.User{Username: "editor1", Email: "editor1@test", Status: models.StatusActive}
	denied := models.User{Username: "editor2", Email: "editor2@test", Status: models.StatusActive}
	for _, u := range []*models.User{&allowed, &denied} {
		if err := db.Create(u).Error; err != nil {
			t.Fatal(err)
		}
	}
	if err := db.Model(&allowed).Association("Groups").Replace([]models.Group{editGroup, adminGroup}); err != nil {
		t.Fatal(err)
	}
	if err := db.Model(&denied).Association("Groups").Replace([]models.Group{adminGroup}); err != nil {
		t.Fatal(err)
	}

	item := models.Item{Title: "Test", Slug: "test-item", CategoryID: &cat.CategoryID, Status: models.ItemStatusDraft}
	if err := db.Create(&item).Error; err != nil {
		t.Fatal(err)
	}

	ok, err := content.CanEditItem(ctx, allowed.UserID, &item)
	if err != nil || !ok {
		t.Fatalf("allowed editor: ok=%v err=%v", ok, err)
	}
	ok, err = content.CanEditItem(ctx, denied.UserID, &item)
	if err != nil || ok {
		t.Fatalf("denied editor: ok=%v err=%v", ok, err)
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
	if err := groups.EnsureDefaults(db); err != nil {
		t.Fatal(err)
	}
	return sites.WithContext(context.Background(), site), db
}
