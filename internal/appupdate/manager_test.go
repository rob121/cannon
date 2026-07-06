package appupdate

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/rob121/cannon/internal/config"
	"github.com/rob121/cannon/internal/version"
)

func TestCheckStoresLatestVersion(t *testing.T) {
	root := t.TempDir()
	platform := runtime.GOOS + "_" + runtime.GOARCH
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/latest/download/cannon.json":
			_, _ = w.Write([]byte(`{
				"name": "cannon",
				"version": "v9.9.9",
				"assets": {
					"` + platform + `": {
						"url": "http://example.test/cannon-binary",
						"sha256": "abc123"
					}
				}
			}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	version.Version = "v0.1.0"
	cfg := &config.App{DataRoot: root, Update: config.UpdateConfig{URLBase: server.URL}}
	mgr := New(cfg)
	mgr.client.HTTP = server.Client()
	mgr.binaryPath = filepath.Join(root, "cannon")

	if err := mgr.Check(); err != nil {
		t.Fatal(err)
	}
	state, err := mgr.LoadState()
	if err != nil {
		t.Fatal(err)
	}
	if !state.UpdateAvailable || state.LatestVersion != "v9.9.9" {
		t.Fatalf("state = %+v", state)
	}
	if state.UpdateAssetURL != "http://example.test/cannon-binary" || state.UpdateAssetSHA256 != "abc123" {
		t.Fatalf("asset = %+v", state)
	}
}

func TestApplyDownloadsAndReplacesBinary(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, "cannon")
	if err := os.WriteFile(target, []byte("old-binary"), 0755); err != nil {
		t.Fatal(err)
	}

	var downloaded bool
	downloadURL := ""
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/latest/download/cannon.json" {
			platform := runtime.GOOS + "_" + runtime.GOARCH
			if downloadURL == "" {
				downloadURL = "http://" + r.Host + "/cannon-binary"
			}
			_, _ = w.Write([]byte(`{
				"name": "cannon",
				"version": "v1.0.0",
				"assets": {
					"` + platform + `": {
						"url": "` + downloadURL + `",
						"sha256": ""
					}
				}
			}`))
			return
		}
		if r.URL.Path == "/cannon-binary" {
			downloaded = true
			_, _ = w.Write([]byte("#!/bin/sh\necho new\n"))
			return
		}
		http.NotFound(w, r)
	}))
	defer server.Close()

	version.Version = "v0.1.0"
	cfg := &config.App{DataRoot: root, Update: config.UpdateConfig{URLBase: server.URL}}
	mgr := New(cfg)
	mgr.client.HTTP = server.Client()
	mgr.binaryPath = target

	if err := mgr.Apply(); err != nil {
		t.Fatal(err)
	}
	if !downloaded {
		t.Fatal("expected update binary download")
	}
	raw, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	if string(raw) != "#!/bin/sh\necho new\n" {
		t.Fatalf("binary = %q", string(raw))
	}
	state, err := mgr.LoadState()
	if err != nil {
		t.Fatal(err)
	}
	if state.UpdateAvailable || !state.RestartRequired {
		t.Fatalf("state = %+v", state)
	}
}
