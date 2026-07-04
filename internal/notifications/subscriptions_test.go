package notifications

import (
	"context"
	"testing"

	"github.com/rob121/cannon/internal/config"
	"github.com/rob121/cannon/internal/database"
	"github.com/rob121/cannon/internal/hooks"
	"github.com/rob121/cannon/internal/models"
	"github.com/rob121/cannon/internal/sites"
	"gorm.io/gorm"
)

func subscriptionsTestDB(t *testing.T) (*gorm.DB, context.Context) {
	t.Helper()
	path := t.TempDir() + "/notifications.sqlite"
	site := &config.SiteConfig{
		ID: t.Name(),
		Database: config.DatabaseConfig{Type: "sqlite", DSN: path},
	}
	if err := database.Migrate(site); err != nil {
		t.Fatal(err)
	}
	db, err := database.Get(site.ID)
	if err != nil {
		t.Fatal(err)
	}
	return db, sites.WithContext(context.Background(), site)
}

func createTestUser(t *testing.T, db *gorm.DB, username, email string) models.User {
	t.Helper()
	u := models.User{Username: username, Email: email, Status: models.StatusActive}
	if err := db.Create(&u).Error; err != nil {
		t.Fatal(err)
	}
	return u
}

func assignRole(t *testing.T, db *gorm.DB, userID, roleID uint) {
	t.Helper()
	if err := db.Exec("INSERT INTO user_roles (user_user_id, role_role_id) VALUES (?, ?)", userID, roleID).Error; err != nil {
		t.Fatal(err)
	}
}

func TestLoadUserProfileStateRoleDefaults(t *testing.T) {
	db, ctx := subscriptionsTestDB(t)
	role := models.Role{Name: "editor", Status: models.StatusActive}
	if err := db.Create(&role).Error; err != nil {
		t.Fatal(err)
	}
	user := createTestUser(t, db, "alice", "alice@example.com")
	assignRole(t, db, user.UserID, role.RoleID)
	if err := ReplaceRoleSubscriptions(db, role.RoleID, []string{hooks.OnItemAfterSave}); err != nil {
		t.Fatal(err)
	}

	state, err := LoadUserProfileState(ctx, user.UserID)
	if err != nil {
		t.Fatal(err)
	}
	if !state.Checked[hooks.OnItemAfterSave] {
		t.Fatal("expected role default event to be checked")
	}
	if !state.RoleDefaults[hooks.OnItemAfterSave] {
		t.Fatal("expected role default marker")
	}
	if state.Checked[hooks.OnUserSignup] {
		t.Fatal("unexpected signup checked")
	}
}

func TestSaveUserSubscriptionsOptOutRoleDefault(t *testing.T) {
	db, ctx := subscriptionsTestDB(t)
	role := models.Role{Name: "editor", Status: models.StatusActive}
	if err := db.Create(&role).Error; err != nil {
		t.Fatal(err)
	}
	user := createTestUser(t, db, "bob", "bob@example.com")
	assignRole(t, db, user.UserID, role.RoleID)
	if err := ReplaceRoleSubscriptions(db, role.RoleID, []string{hooks.OnItemAfterSave}); err != nil {
		t.Fatal(err)
	}
	if err := SaveUserSubscriptions(ctx, user.UserID, nil); err != nil {
		t.Fatal(err)
	}

	state, err := LoadUserProfileState(ctx, user.UserID)
	if err != nil {
		t.Fatal(err)
	}
	if state.Checked[hooks.OnItemAfterSave] {
		t.Fatal("expected opt-out to uncheck role default")
	}

	var row models.NotificationSubscription
	err = db.Where("user_id = ? AND event = ?", user.UserID, hooks.OnItemAfterSave).First(&row).Error
	if err != nil {
		t.Fatal(err)
	}
	if row.Status != models.StatusInactive {
		t.Fatalf("expected inactive opt-out, got %s", row.Status)
	}
}

func TestSaveUserSubscriptionsOptInWithoutRoleDefault(t *testing.T) {
	db, ctx := subscriptionsTestDB(t)
	user := createTestUser(t, db, "carol", "carol@example.com")
	if err := SaveUserSubscriptions(ctx, user.UserID, []string{hooks.OnCommentAfterSave}); err != nil {
		t.Fatal(err)
	}

	state, err := LoadUserProfileState(ctx, user.UserID)
	if err != nil {
		t.Fatal(err)
	}
	if !state.Checked[hooks.OnCommentAfterSave] {
		t.Fatal("expected user opt-in to be checked")
	}
}

func TestResolveEmailRecipients(t *testing.T) {
	db, ctx := subscriptionsTestDB(t)
	role := models.Role{Name: "staff", Status: models.StatusActive}
	if err := db.Create(&role).Error; err != nil {
		t.Fatal(err)
	}
	roleUser := createTestUser(t, db, "roleuser", "role@example.com")
	optOutUser := createTestUser(t, db, "optout", "optout@example.com")
	optInUser := createTestUser(t, db, "optin", "optin@example.com")
	assignRole(t, db, roleUser.UserID, role.RoleID)
	assignRole(t, db, optOutUser.UserID, role.RoleID)
	if err := ReplaceRoleSubscriptions(db, role.RoleID, []string{hooks.OnUserSignup}); err != nil {
		t.Fatal(err)
	}
	if err := SaveUserSubscriptions(ctx, optOutUser.UserID, nil); err != nil {
		t.Fatal(err)
	}
	if err := SaveUserSubscriptions(ctx, optInUser.UserID, []string{hooks.OnUserSignup}); err != nil {
		t.Fatal(err)
	}

	recipients, err := resolveEmailRecipients(ctx, hooks.OnUserSignup)
	if err != nil {
		t.Fatal(err)
	}
	emails := map[string]struct{}{}
	for _, rcpt := range recipients {
		emails[rcpt.Email] = struct{}{}
	}
	if _, ok := emails["role@example.com"]; !ok {
		t.Fatal("expected role member recipient")
	}
	if _, ok := emails["optin@example.com"]; !ok {
		t.Fatal("expected opt-in recipient")
	}
	if _, ok := emails["optout@example.com"]; ok {
		t.Fatal("expected opt-out user to be excluded")
	}
}

func TestReplaceRoleSubscriptionsClearsPrevious(t *testing.T) {
	db, _ := subscriptionsTestDB(t)
	role := models.Role{Name: "temp", Status: models.StatusActive}
	if err := db.Create(&role).Error; err != nil {
		t.Fatal(err)
	}
	if err := ReplaceRoleSubscriptions(db, role.RoleID, []string{hooks.OnUserSignup, hooks.OnUserLogout}); err != nil {
		t.Fatal(err)
	}
	if err := ReplaceRoleSubscriptions(db, role.RoleID, []string{hooks.OnUserVerified}); err != nil {
		t.Fatal(err)
	}
	events, err := RoleSubscriptionEvents(db, role.RoleID)
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 1 || events[0] != hooks.OnUserVerified {
		t.Fatalf("unexpected events: %v", events)
	}
}
