package notifications

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/rob121/cannon/internal/config"
	"github.com/rob121/cannon/internal/database"
	"github.com/rob121/cannon/internal/models"
	"github.com/rob121/cannon/internal/sites"
)

func TestRecordDeliverySuccessAndFailure(t *testing.T) {
	site := &config.SiteConfig{
		ID: t.Name(),
		Database: config.DatabaseConfig{
			Type: "sqlite",
			DSN:  filepath.Join(t.TempDir(), "site.db"),
		},
	}
	if err := database.Migrate(site); err != nil {
		t.Fatal(err)
	}
	db, err := database.Get(site.ID)
	if err != nil {
		t.Fatal(err)
	}
	_ = sites.WithContext(context.Background(), site)

	row := models.Notification{Name: "Slack", ShoutrrURL: "https://example.com/hook", Status: models.StatusActive}
	if err := db.Create(&row).Error; err != nil {
		t.Fatal(err)
	}
	if err := recordDeliveryFailure(db, row.NotificationID, "boom"); err != nil {
		t.Fatal(err)
	}
	var failed models.Notification
	if err := db.First(&failed, row.NotificationID).Error; err != nil {
		t.Fatal(err)
	}
	if failed.LastError != "boom" || failed.LastErrorAt == nil {
		t.Fatalf("failure not recorded: %+v", failed)
	}
	if err := recordDeliverySuccess(db, row.NotificationID); err != nil {
		t.Fatal(err)
	}
	var okRow models.Notification
	if err := db.First(&okRow, row.NotificationID).Error; err != nil {
		t.Fatal(err)
	}
	if okRow.LastError != "" || okRow.LastSentAt == nil {
		t.Fatalf("success not recorded: %+v", okRow)
	}
	if time.Since(*okRow.LastSentAt) > time.Minute {
		t.Fatalf("unexpected last_sent_at: %v", okRow.LastSentAt)
	}
}

func TestSendTestRecordsFailure(t *testing.T) {
	site := &config.SiteConfig{
		ID: t.Name(),
		Database: config.DatabaseConfig{
			Type: "sqlite",
			DSN:  filepath.Join(t.TempDir(), "site.db"),
		},
	}
	if err := database.Migrate(site); err != nil {
		t.Fatal(err)
	}
	db, err := database.Get(site.ID)
	if err != nil {
		t.Fatal(err)
	}
	ctx := sites.WithContext(context.Background(), site)

	row := models.Notification{Name: "Broken", ShoutrrURL: "not-a-valid-shoutrrr-url", Status: models.StatusActive}
	if err := db.Create(&row).Error; err != nil {
		t.Fatal(err)
	}
	if err := SendTest(ctx, row.NotificationID); err == nil {
		t.Fatal("expected SendTest to fail")
	}
	var saved models.Notification
	if err := db.First(&saved, row.NotificationID).Error; err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(saved.LastError) == "" || saved.LastErrorAt == nil {
		t.Fatalf("expected last error after test: %+v", saved)
	}
}
