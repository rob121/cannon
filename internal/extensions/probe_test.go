package extensions

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/rob121/cannon/internal/config"
	"github.com/rob121/cannon/internal/database"
	"github.com/rob121/cannon/internal/models"
	"github.com/rob121/cannon/internal/sites"
)

func TestParseBinaryMetaOutput(t *testing.T) {
	raw := `defaultAdminMenu=Drive Backup
extensionDescription=Backs up Cannon data_root.
extensionName=cannon-ext-backup-gdrive
extensionTitle=Google Drive Backup
extensionVersion=0.1.1
updateURLBase=https://github.com/rob121/cannon-ext-backup-gdrive/releases/download
`
	meta, err := parseBinaryMetaOutput(raw)
	if err != nil {
		t.Fatal(err)
	}
	if meta.Name != "cannon-ext-backup-gdrive" ||
		meta.Title != "Google Drive Backup" ||
		meta.Description != "Backs up Cannon data_root." ||
		meta.Version != "0.1.1" ||
		meta.UpdateURLBase != "https://github.com/rob121/cannon-ext-backup-gdrive/releases/download" ||
		meta.AdminMenuName != "Drive Backup" {
		t.Fatalf("meta = %+v", meta)
	}
}

func TestProbeBinaryMetaAndEnrich(t *testing.T) {
	root := t.TempDir()
	extDir := filepath.Join(root, "extensions")
	if err := os.MkdirAll(extDir, 0755); err != nil {
		t.Fatal(err)
	}
	script := `#!/bin/sh
if [ "$1" = "--data=all" ]; then
  cat <<'EOF'
defaultAdminMenu=Drive Backup
extensionDescription=Backs up Cannon data_root and sites.json to Google Drive.
extensionName=cannon-ext-backup-gdrive
extensionTitle=Google Drive Backup
extensionVersion=0.1.1
updateURLBase=https://github.com/rob121/cannon-ext-backup-gdrive/releases/download
EOF
  exit 0
fi
exit 1
`
	bin := filepath.Join(extDir, "cannon-ext-backup-gdrive")
	if err := os.WriteFile(bin, []byte(script), 0755); err != nil {
		t.Fatal(err)
	}

	site := &config.SiteConfig{ID: t.Name(), Database: config.DatabaseConfig{Type: "sqlite", DSN: filepath.Join(root, "site.db")}}
	db, err := database.Connect(site)
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&models.Extension{}); err != nil {
		t.Fatal(err)
	}
	app := &config.App{}
	app.Extensions.Dir = extDir
	app.Extensions.SocketsDir = filepath.Join(root, "sockets")
	mgr := NewManager(app, site)
	ctx := sites.WithContext(context.Background(), site)

	probed, err := mgr.ProbeBinaryMeta("cannon-ext-backup-gdrive")
	if err != nil {
		t.Fatal(err)
	}
	if probed.Title != "Google Drive Backup" || probed.Version != "0.1.1" {
		t.Fatalf("probed = %+v", probed)
	}

	if err := mgr.syncDirectory(db); err != nil {
		t.Fatal(err)
	}
	var before models.Extension
	if err := db.Where("name = ?", "cannon-ext-backup-gdrive").First(&before).Error; err != nil {
		t.Fatal(err)
	}
	if before.Title != "" || before.Installed {
		t.Fatalf("expected empty metadata before enrich, got %+v", before)
	}

	if err := mgr.enrichBinaryMeta(ctx, db); err != nil {
		t.Fatal(err)
	}
	var after models.Extension
	if err := db.Where("name = ?", "cannon-ext-backup-gdrive").First(&after).Error; err != nil {
		t.Fatal(err)
	}
	if after.Title != "Google Drive Backup" ||
		after.Description == "" ||
		after.Version != "0.1.1" ||
		after.UpdateURLBase == "" ||
		after.MenuName != "Drive Backup" ||
		after.Installed {
		t.Fatalf("after enrich = %+v", after)
	}
}

func TestEnrichBinaryMetaOverwritesStaleUpdateURLBase(t *testing.T) {
	root := t.TempDir()
	extDir := filepath.Join(root, "extensions")
	if err := os.MkdirAll(extDir, 0755); err != nil {
		t.Fatal(err)
	}
	script := `#!/bin/sh
if [ "$1" = "--data=all" ]; then
  cat <<'EOF'
extensionName=cannon-ext-powwow
extensionTitle=Pow Wow
extensionDescription=Event dates.
extensionVersion=1.2.1
updateURLBase=https://github.com/floridarangers/cannon-ext-powwow/releases/download
defaultAdminMenu=Pow Wow
EOF
  exit 0
fi
exit 1
`
	bin := filepath.Join(extDir, "cannon-ext-powwow")
	if err := os.WriteFile(bin, []byte(script), 0755); err != nil {
		t.Fatal(err)
	}

	site := &config.SiteConfig{ID: t.Name(), Database: config.DatabaseConfig{Type: "sqlite", DSN: filepath.Join(root, "site.db")}}
	db, err := database.Connect(site)
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&models.Extension{}); err != nil {
		t.Fatal(err)
	}
	app := &config.App{}
	app.Extensions.Dir = extDir
	app.Extensions.SocketsDir = filepath.Join(root, "sockets")
	mgr := NewManager(app, site)
	ctx := sites.WithContext(context.Background(), site)

	if err := db.Create(&models.Extension{
		Name:          "cannon-ext-powwow",
		Socket:        mgr.socketPath("cannon-ext-powwow"),
		Status:        models.StatusInactive,
		Title:         "Pow Wow",
		Description:   "Event dates.",
		Version:       "1.2.0",
		UpdateURLBase: "https://github.com/rob121/cannon-ext-powwow/releases/download",
		UpdateError:   "no update manifest or GitHub latest release found",
		MenuName:      "Pow Wow",
		Installed:     true,
	}).Error; err != nil {
		t.Fatal(err)
	}

	if err := mgr.enrichBinaryMeta(ctx, db); err != nil {
		t.Fatal(err)
	}
	var after models.Extension
	if err := db.Where("name = ?", "cannon-ext-powwow").First(&after).Error; err != nil {
		t.Fatal(err)
	}
	want := "https://github.com/floridarangers/cannon-ext-powwow/releases/download"
	if after.UpdateURLBase != want {
		t.Fatalf("UpdateURLBase = %q, want %q", after.UpdateURLBase, want)
	}
	if after.Version != "1.2.1" {
		t.Fatalf("Version = %q, want 1.2.1", after.Version)
	}
	if after.UpdateError != "" {
		t.Fatalf("UpdateError still set: %q", after.UpdateError)
	}
}
