package security

import (
	"testing"

	"github.com/rob121/cannon/internal/models"
)

func TestParentRoleDenyDoesNotPropagateToChildRoleUser(t *testing.T) {
	ctx, db := testContext(t)

	const stripeSetup = "cannon-ext-stripe-subscription.stripe.setup"

	var editor, administrator models.Role
	if err := db.Where("name = ?", RoleEditor).First(&editor).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Where("name = ?", RoleAdministrator).First(&administrator).Error; err != nil {
		t.Fatal(err)
	}
	if err := SaveRolePermissions(db, editor.RoleID, nil, []string{stripeSetup}); err != nil {
		t.Fatal(err)
	}

	adminUser := models.User{Username: "siteadmin", Email: "siteadmin@test", Status: models.StatusActive}
	if err := db.Create(&adminUser).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Model(&adminUser).Association("Roles").Append(&administrator); err != nil {
		t.Fatal(err)
	}

	ok, err := Can(ctx, adminUser.UserID, stripeSetup)
	if err != nil || !ok {
		t.Fatalf("administrator user should not inherit editor role deny: ok=%v err=%v", ok, err)
	}

	editorUser := models.User{Username: "editoronly", Email: "editoronly@test", Status: models.StatusActive}
	if err := db.Create(&editorUser).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Model(&editorUser).Association("Roles").Append(&editor); err != nil {
		t.Fatal(err)
	}
	ok, err = Can(ctx, editorUser.UserID, stripeSetup)
	if err != nil || ok {
		t.Fatalf("editor user should be denied stripe setup: ok=%v err=%v", ok, err)
	}
}

func TestParentPermissionKeysOmitsParentDenies(t *testing.T) {
	_, db := testContext(t)
	const stripeSetup = "cannon-ext-stripe-subscription.stripe.setup"

	var editor, manager models.Role
	if err := db.Where("name = ?", RoleEditor).First(&editor).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Where("name = ?", RoleManager).First(&manager).Error; err != nil {
		t.Fatal(err)
	}
	if err := SaveRolePermissions(db, editor.RoleID, nil, []string{stripeSetup}); err != nil {
		t.Fatal(err)
	}

	allows, denies, err := ParentPermissionKeys(db, []uint{editor.RoleID})
	if err != nil {
		t.Fatal(err)
	}
	for _, key := range denies {
		if key == stripeSetup {
			t.Fatal("parent deny should not appear in inherited deny keys for role editor UI")
		}
	}
	_ = allows

	parentAllows, parentDenies, err := ParentPermissionKeys(db, []uint{manager.RoleID})
	if err != nil {
		t.Fatal(err)
	}
	for _, key := range parentDenies {
		if key == stripeSetup {
			t.Fatal("editor deny should not surface when listing manager parent inheritance")
		}
	}
	_ = parentAllows
}

func TestChildRoleDirectDenyOverridesInheritedAllow(t *testing.T) {
	ctx, db := testContext(t)

	var viewer, editor models.Role
	if err := db.Where("name = ?", RoleViewer).First(&viewer).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Where("name = ?", RoleEditor).First(&editor).Error; err != nil {
		t.Fatal(err)
	}
	const perm = "core.view.test"
	if err := SaveRolePermissions(db, viewer.RoleID, []string{perm}, nil); err != nil {
		t.Fatal(err)
	}
	if err := SaveRolePermissions(db, editor.RoleID, nil, []string{perm}); err != nil {
		t.Fatal(err)
	}

	user := models.User{Username: "editor", Email: "editor@test", Status: models.StatusActive}
	if err := db.Create(&user).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Model(&user).Association("Roles").Append(&editor); err != nil {
		t.Fatal(err)
	}

	ok, err := Can(ctx, user.UserID, perm)
	if err != nil || ok {
		t.Fatalf("expected editor direct deny to override inherited viewer allow: ok=%v err=%v", ok, err)
	}
}
