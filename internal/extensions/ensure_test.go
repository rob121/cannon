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

func TestEnsureActiveRuntimeSkipsInactive(t *testing.T) {
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
		Name:   "cannon-ext-demo",
		Socket: mgr.socketPath("cannon-ext-demo"),
		Status: models.StatusInactive,
	}
	if err := db.Create(&row).Error; err != nil {
		t.Fatal(err)
	}

	_, err = mgr.ensureActiveRuntime(ctx, row.Name)
	if err == nil {
		t.Fatal("expected inactive extension to be skipped")
	}
	if mgr.IsRunning(row.Name) {
		t.Fatal("inactive extension should not be running")
	}
}

func TestDisableFailedExtensionKeepsActiveStatus(t *testing.T) {
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

	err = mgr.Start(ctx, row)
	if err == nil {
		t.Fatal("expected start failure for missing binary")
	}
	var saved models.Extension
	if err := db.First(&saved, row.ExtensionID).Error; err != nil {
		t.Fatal(err)
	}
	if saved.Status != models.StatusActive {
		t.Fatalf("status = %q, want active (start failure must not flip status)", saved.Status)
	}
	if !mgr.isSuppressed(row.Name) {
		t.Fatal("expected failed start to suppress bootstrap auto-start")
	}
}
