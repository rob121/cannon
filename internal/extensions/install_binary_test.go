package extensions

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rob121/cannon/internal/config"
	"github.com/rob121/cannon/internal/database"
	"github.com/rob121/cannon/internal/models"
	"github.com/rob121/cannon/internal/sites"
)

func TestSaveBinaryWritesExecutableAndRegistersRow(t *testing.T) {
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

	row, err := mgr.SaveBinary(ctx, SaveBinaryOptions{
		Name:          "cannon-ext-demo",
		Source:        strings.NewReader("#!/bin/sh\nexit 0\n"),
		UpdateURLBase: "https://github.com/rob121/cannon-ext-demo/releases/download",
		LatestVersion: "v0.1.0",
	})
	if err != nil {
		t.Fatal(err)
	}
	if row.Name != "cannon-ext-demo" || row.Socket == "" || row.UpdateURLBase == "" || row.LatestVersion != "v0.1.0" {
		t.Fatalf("row = %+v", row)
	}
	info, err := os.Stat(filepath.Join(app.Extensions.Dir, "cannon-ext-demo"))
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm()&0111 == 0 {
		t.Fatalf("expected executable mode, got %v", info.Mode().Perm())
	}
}

func TestSaveBinaryRejectsUnsafeName(t *testing.T) {
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

	if _, err := mgr.SaveBinary(ctx, SaveBinaryOptions{Name: "../bad name", Source: strings.NewReader("x")}); err == nil {
		t.Fatal("expected unsafe name to be rejected")
	}
}
