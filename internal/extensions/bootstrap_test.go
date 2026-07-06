package extensions

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/rob121/cannon/internal/config"
	"github.com/rob121/cannon/internal/database"
	"github.com/rob121/cannon/internal/models"
	"github.com/rob121/cannon/internal/sites"
)

func TestBootstrapDisablesMissingBinary(t *testing.T) {
	root := t.TempDir()
	site := &config.SiteConfig{ID: t.Name(), Database: config.DatabaseConfig{Type: "sqlite", DSN: filepath.Join(root, "site.db")}}
	db, err := database.Connect(site)
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&models.Extension{}); err != nil {
		t.Fatal(err)
	}
	app := &config.App{}
	app.Extensions.Dir = filepath.Join(root, "extensions")
	app.Extensions.SocketsDir = filepath.Join(root, "sockets")
	mgr := NewManager(app, site)
	ctx := sites.WithContext(context.Background(), site)

	row := models.Extension{
		Name:   "cannon-ext-missing",
		Socket: mgr.socketPath("cannon-ext-missing"),
		Status: models.StatusActive,
	}
	if err := db.Create(&row).Error; err != nil {
		t.Fatal(err)
	}

	if err := mgr.Bootstrap(ctx); err != nil {
		t.Fatal(err)
	}
	if !mgr.isSuppressed("cannon-ext-missing") {
		t.Fatal("expected missing binary to be suppressed")
	}
	var saved models.Extension
	if err := db.First(&saved, row.ExtensionID).Error; err != nil {
		t.Fatal(err)
	}
	if saved.Status != models.StatusInactive {
		t.Fatalf("status = %q, want inactive", saved.Status)
	}
	if mgr.IsRunning("cannon-ext-missing") {
		t.Fatal("expected extension not to be running")
	}
}
