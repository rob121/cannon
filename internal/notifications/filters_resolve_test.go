package notifications

import (
	"testing"

	"github.com/rob121/cannon/internal/hooks"
	"github.com/rob121/cannon/internal/models"
	"github.com/rob121/cannon/internal/security"
)

func TestResolveEmailRecipientsAppliesRoleFilters(t *testing.T) {
	db, ctx := subscriptionsTestDB(t) // db, ctx order matches helper
	role := models.Role{Name: "editors", Status: models.StatusActive}
	if err := db.Create(&role).Error; err != nil {
		t.Fatal(err)
	}
	user := createTestUser(t, db, "editor", "editor@example.com")
	assignRole(t, db, user.UserID, role.RoleID)
	if err := security.SaveRolePermissions(db, role.RoleID, []string{security.PermAdminAccess}, nil); err != nil {
		t.Fatal(err)
	}
	if err := ReplaceRoleSubscriptionsWithFilters(db, role.RoleID, []string{hooks.OnItemAfterSave}, map[string]SubscriptionFilters{
		hooks.OnItemAfterSave: {Status: "pending"},
	}); err != nil {
		t.Fatal(err)
	}

	author := user.UserID
	pending := models.Item{Status: models.ItemStatusPending, AuthorID: &author}
	draft := models.Item{Status: models.ItemStatusDraft, AuthorID: &author}

	got, err := resolveEmailRecipients(ctx, hooks.OnItemAfterSave, map[string]any{"item": pending})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].Email != "editor@example.com" {
		t.Fatalf("pending recipients = %+v", got)
	}

	got, err = resolveEmailRecipients(ctx, hooks.OnItemAfterSave, map[string]any{"item": draft})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 0 {
		t.Fatalf("draft should filter out, got %+v", got)
	}
}
