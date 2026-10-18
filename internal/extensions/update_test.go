package extensions

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/rob121/cannon/internal/config"
	"github.com/rob121/cannon/internal/database"
	"github.com/rob121/cannon/internal/models"
	"github.com/rob121/cannon/internal/sites"
	"github.com/rob121/cannon/internal/updater"
)

func TestNewerVersion(t *testing.T) {
	if !updater.NewerVersion("v0.2.0", "0.1.0") {
		t.Fatal("expected v0.2.0 to be newer than 0.1.0")
	}
	if updater.NewerVersion("v0.1.0", "0.1.0") {
		t.Fatal("expected equivalent versions to not be newer")
	}
}

func TestGitHubLatestAPIURL(t *testing.T) {
	got := updater.GitHubLatestAPIURL("https://github.com/rob121/cannon-ext-gcalendar/releases/download")
	want := "https://api.github.com/repos/rob121/cannon-ext-gcalendar/releases/latest"
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
	if got := updater.GitHubLatestAPIURL("https://example.com/releases/download"); got != "" {
		t.Fatalf("expected non-GitHub URL to be ignored, got %q", got)
	}
}

func TestUpdateManifestURLForGitHubReleaseBase(t *testing.T) {
	got := updater.ManifestURL("https://github.com/rob121/cannon-ext-gzip/releases/download", updateManifestName)
	want := "https://github.com/rob121/cannon-ext-gzip/releases/latest/download/cannon-extension.json"
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestLatestUpdateInfoFromManifest(t *testing.T) {
	platform := runtime.GOOS + "_" + runtime.GOARCH
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/latest/download/cannon-extension.json" {
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
		_, _ = w.Write([]byte(`{
			"name": "demo-extension",
			"version": "0.2.0",
			"assets": {
				"` + platform + `": {
					"url": "http://example.test/demo-extension",
					"sha256": "sha256:abc123"
				}
			}
		}`))
	}))
	defer server.Close()

	m := NewManager(nil, nil)
	m.updateClient = server.Client()
	info, err := m.latestUpdateInfo(models.Extension{Name: "demo-extension", UpdateURLBase: server.URL})
	if err != nil {
		t.Fatal(err)
	}
	if info.Version != "0.2.0" || info.URL != "http://example.test/demo-extension" || info.SHA256 != "abc123" {
		t.Fatalf("unexpected update info: %+v", info)
	}
}

func TestUpdateAllNoOutdated(t *testing.T) {
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

	if err := db.Create(&models.Extension{
		Name:            "current-ext",
		Socket:          mgr.socketPath("current-ext"),
		Status:          models.StatusInactive,
		Version:         "1.0.0",
		UpdateAvailable: false,
	}).Error; err != nil {
		t.Fatal(err)
	}

	result, err := mgr.UpdateAll(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if result.Updated != 0 || result.Failed != 0 || len(result.Errors) != 0 {
		t.Fatalf("result = %+v", result)
	}
}

func TestUpdateAllUpdatesOutdated(t *testing.T) {
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
	if err := os.MkdirAll(app.Extensions.Dir, 0o755); err != nil {
		t.Fatal(err)
	}
	mgr := NewManager(app, site)
	ctx := sites.WithContext(context.Background(), site)

	platform := runtime.GOOS + "_" + runtime.GOARCH
	payload := []byte("#!/bin/sh\necho updated\n")
	digest := hex.EncodeToString(sha256Sum(payload))

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasSuffix(r.URL.Path, "/latest/download/cannon-extension.json"):
			name := strings.TrimPrefix(strings.TrimSuffix(r.URL.Path, "/latest/download/cannon-extension.json"), "/")
			_, _ = w.Write([]byte(`{
				"name": "` + name + `",
				"version": "0.2.0",
				"assets": {
					"` + platform + `": {
						"url": "http://` + r.Host + `/` + name + `/bin",
						"sha256": "sha256:` + digest + `"
					}
				}
			}`))
		case strings.HasSuffix(r.URL.Path, "/bin"):
			_, _ = w.Write(payload)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	mgr.updateClient = server.Client()

	for _, name := range []string{"ext-a", "ext-b"} {
		binPath := filepath.Join(app.Extensions.Dir, name)
		if err := os.WriteFile(binPath, []byte("#!/bin/sh\necho old\n"), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := db.Create(&models.Extension{
			Name:              name,
			Socket:            mgr.socketPath(name),
			Status:            models.StatusInactive,
			Installed:         true,
			Version:           "0.1.0",
			LatestVersion:     "0.2.0",
			UpdateURLBase:     server.URL + "/" + name,
			UpdateAvailable:   true,
			UpdateAssetURL:    server.URL + "/" + name + "/bin",
			UpdateAssetSHA256: digest,
		}).Error; err != nil {
			t.Fatal(err)
		}
	}
	if err := db.Create(&models.Extension{
		Name:            "ext-current",
		Socket:          mgr.socketPath("ext-current"),
		Status:          models.StatusInactive,
		Installed:       true,
		Version:         "0.2.0",
		UpdateAvailable: false,
	}).Error; err != nil {
		t.Fatal(err)
	}

	result, err := mgr.UpdateAll(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if result.Updated != 2 || result.Failed != 0 {
		t.Fatalf("result = %+v", result)
	}

	var outdated int64
	if err := db.Model(&models.Extension{}).Where("update_available = ?", true).Count(&outdated).Error; err != nil {
		t.Fatal(err)
	}
	if outdated != 0 {
		t.Fatalf("expected no outdated extensions, got %d", outdated)
	}
	for _, name := range []string{"ext-a", "ext-b"} {
		got, err := os.ReadFile(filepath.Join(app.Extensions.Dir, name))
		if err != nil {
			t.Fatal(err)
		}
		if string(got) != string(payload) {
			t.Fatalf("%s binary not updated", name)
		}
		var row models.Extension
		if err := db.Where("name = ?", name).First(&row).Error; err != nil {
			t.Fatal(err)
		}
		if row.Version != "0.2.0" || row.UpdateAvailable {
			t.Fatalf("%s row = %+v", name, row)
		}
	}
}

func sha256Sum(b []byte) []byte {
	sum := sha256.Sum256(b)
	return sum[:]
}
