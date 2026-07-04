package api

import (
	"context"
	"testing"
	"time"

	"github.com/rob121/cannon/internal/config"
	"github.com/rob121/cannon/internal/database"
	"github.com/rob121/cannon/internal/models"
	"github.com/rob121/cannon/internal/sites"
)

func apiTestCtx(t *testing.T) context.Context {
	t.Helper()
	path := t.TempDir() + "/api.sqlite"
	site := &config.SiteConfig{
		ID: t.Name(),
		Database: config.DatabaseConfig{Type: "sqlite", DSN: path},
	}
	if err := database.Migrate(site); err != nil {
		t.Fatal(err)
	}
	return sites.WithContext(context.Background(), site)
}

func TestIssueAndValidateAPIKey(t *testing.T) {
	ctx := apiTestCtx(t)
	row, token, err := IssueCredential(ctx, "Test App", nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if row.CredentialID == 0 || token == "" {
		t.Fatal("expected credential and token")
	}
	got, err := ValidateAPIKey(ctx, token)
	if err != nil {
		t.Fatal(err)
	}
	if got.CredentialID != row.CredentialID {
		t.Fatalf("credential id mismatch: %d vs %d", got.CredentialID, row.CredentialID)
	}
	if _, err := ValidateAPIKey(ctx, token+"x"); err == nil {
		t.Fatal("expected invalid key error")
	}
}

func TestJWTIssueAndParse(t *testing.T) {
	ctx := apiTestCtx(t)
	db, err := sites.DB(ctx)
	if err != nil {
		t.Fatal(err)
	}
	u := models.User{Username: "apiuser", Email: "api@example.com", Status: models.StatusActive, Validated: true}
	if err := db.Create(&u).Error; err != nil {
		t.Fatal(err)
	}
	token, exp, err := IssueAccessToken(ctx, u.UserID)
	if err != nil {
		t.Fatal(err)
	}
	if token == "" || exp <= 0 {
		t.Fatal("expected access token")
	}
	parsedID, err := ParseAccessToken(ctx, token)
	if err != nil {
		t.Fatal(err)
	}
	if parsedID != u.UserID {
		t.Fatalf("user id %d want %d", parsedID, u.UserID)
	}
}

func TestPendingTokenRoundTrip(t *testing.T) {
	ctx := apiTestCtx(t)
	raw, err := IssuePendingToken(ctx, PendingMFALogin, 42, map[string]string{"k": "v"}, time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	row, err := ConsumePendingToken(ctx, PendingMFALogin, raw)
	if err != nil || row.UserID != 42 {
		t.Fatalf("consume: %v user=%d", err, row.UserID)
	}
	if _, err := ConsumePendingToken(ctx, PendingMFALogin, raw); err == nil {
		t.Fatal("expected token reuse failure")
	}
}
