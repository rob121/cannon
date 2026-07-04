package user

import (
	"context"
	"testing"

	"github.com/rob121/cannon/internal/config"
	"github.com/rob121/cannon/internal/database"
	"github.com/rob121/cannon/internal/models"
	"github.com/rob121/cannon/internal/sites"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

func profileTestDB(t *testing.T) (*gorm.DB, context.Context) {
	t.Helper()
	path := t.TempDir() + "/profile.sqlite"
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

func TestIsUsernameAvailable(t *testing.T) {
	db, _ := profileTestDB(t)
	hash, _ := bcrypt.GenerateFromPassword([]byte("password"), bcrypt.DefaultCost)
	u := models.User{Username: "alice", Email: "a@example.com", Status: models.StatusActive, Hash: string(hash)}
	if err := db.Create(&u).Error; err != nil {
		t.Fatal(err)
	}
	ok, err := IsUsernameAvailable(db, "alice", u.UserID)
	if err != nil || !ok {
		t.Fatalf("own username: ok=%v err=%v", ok, err)
	}
	ok, err = IsUsernameAvailable(db, "alice", 0)
	if err != nil || ok {
		t.Fatalf("other user: ok=%v err=%v", ok, err)
	}
	ok, err = IsUsernameAvailable(db, "bob", 0)
	if err != nil || !ok {
		t.Fatalf("free username: ok=%v err=%v", ok, err)
	}
}

func TestUpdateProfileIdentity(t *testing.T) {
	db, ctx := profileTestDB(t)
	hash, _ := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.DefaultCost)
	u := models.User{Username: "jane", Email: "jane@example.com", Status: models.StatusActive, Hash: string(hash)}
	if err := db.Create(&u).Error; err != nil {
		t.Fatal(err)
	}
	if err := UpdateProfileIdentity(ctx, u.UserID, "jane2", "jane2@example.com"); err != nil {
		t.Fatal(err)
	}
	var updated models.User
	if err := db.First(&updated, u.UserID).Error; err != nil {
		t.Fatal(err)
	}
	if updated.Username != "jane2" || updated.Email != "jane2@example.com" {
		t.Fatalf("unexpected update: %+v", updated)
	}
}
