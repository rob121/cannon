package appupdate

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

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
	mgr.SetBinaryPath(filepath.Join(root, "cannon"))

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

func TestApplyStagesLatestWithoutReplacingLive(t *testing.T) {
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
	mgr.SetBinaryPath(target)

	if err := mgr.Apply(); err != nil {
		t.Fatal(err)
	}
	if !downloaded {
		t.Fatal("expected update binary download")
	}
	live, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	if string(live) != "old-binary" {
		t.Fatalf("live binary changed before restart: %q", string(live))
	}
	staged, err := os.ReadFile(latestPath(target))
	if err != nil {
		t.Fatal(err)
	}
	if string(staged) != "#!/bin/sh\necho new\n" {
		t.Fatalf("staged = %q", string(staged))
	}
	state, err := mgr.LoadState()
	if err != nil {
		t.Fatal(err)
	}
	if !state.RestartRequired || state.StagedVersion != "v1.0.0" {
		t.Fatalf("state = %+v", state)
	}
}

func TestApplyPendingSwapsLatestAndKeepsPrevious(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, "cannon")
	latest := latestPath(target)
	if err := os.WriteFile(target, []byte("old-binary"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(latest, []byte("new-binary"), 0755); err != nil {
		t.Fatal(err)
	}
	applied, err := applyPending(target)
	if err != nil {
		t.Fatal(err)
	}
	if !applied {
		t.Fatal("expected pending apply")
	}
	live, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	if string(live) != "new-binary" {
		t.Fatalf("live = %q", string(live))
	}
	prev, err := os.ReadFile(previousPath(target))
	if err != nil {
		t.Fatal(err)
	}
	if string(prev) != "old-binary" {
		t.Fatalf("previous = %q", string(prev))
	}
	if fileExists(latest) {
		t.Fatal("expected .latest removed after apply")
	}
}

func TestApplyPendingRepairsMistakenPreviousLatest(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, "cannon")
	mistaken := filepath.Join(root, "cannon.previous.latest")
	if err := os.WriteFile(target, []byte("old-binary"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(mistaken, []byte("new-binary"), 0755); err != nil {
		t.Fatal(err)
	}
	applied, err := applyPending(filepath.Join(root, "cannon.previous"))
	if err != nil {
		t.Fatal(err)
	}
	if !applied {
		t.Fatal("expected pending apply after repairing mistaken name")
	}
	live, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	if string(live) != "new-binary" {
		t.Fatalf("live = %q", string(live))
	}
	if fileExists(mistaken) {
		t.Fatal("expected mistaken .previous.latest removed")
	}
	if fileExists(latestPath(target)) {
		t.Fatal("expected .latest consumed by apply")
	}
}

func TestRollbackFlipsPrevious(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, "cannon")
	prev := previousPath(target)
	if err := os.WriteFile(target, []byte("new-binary"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(prev, []byte("old-binary"), 0755); err != nil {
		t.Fatal(err)
	}
	cfg := &config.App{DataRoot: root}
	mgr := New(cfg)
	mgr.SetBinaryPath(target)
	if err := mgr.Rollback(); err != nil {
		t.Fatal(err)
	}
	live, _ := os.ReadFile(target)
	demoted, _ := os.ReadFile(prev)
	if string(live) != "old-binary" || string(demoted) != "new-binary" {
		t.Fatalf("live=%q previous=%q", live, demoted)
	}
}

func TestApplyFromPreviousExecutableNameStagesCannonLatest(t *testing.T) {
	root := t.TempDir()
	live := filepath.Join(root, "cannon")
	prev := filepath.Join(root, "cannon.previous")
	if err := os.WriteFile(live, []byte("new-live"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(prev, []byte("old-running"), 0755); err != nil {
		t.Fatal(err)
	}

	downloadURL := ""
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/latest/download/cannon.json" {
			platform := runtime.GOOS + "_" + runtime.GOARCH
			if downloadURL == "" {
				downloadURL = "http://" + r.Host + "/cannon-binary"
			}
			_, _ = w.Write([]byte(`{
				"name": "cannon",
				"version": "v2.0.0",
				"assets": { "` + platform + `": { "url": "` + downloadURL + `", "sha256": "" } }
			}`))
			return
		}
		if r.URL.Path == "/cannon-binary" {
			_, _ = w.Write([]byte("staged-binary"))
			return
		}
		http.NotFound(w, r)
	}))
	defer server.Close()

	version.Version = "v1.0.0"
	mgr := New(&config.App{DataRoot: root, Update: config.UpdateConfig{URLBase: server.URL}})
	mgr.client.HTTP = server.Client()
	// Simulate Linux after swap: Executable() points at the renamed inode.
	mgr.SetBinaryPath(prev)

	if err := mgr.Apply(); err != nil {
		t.Fatal(err)
	}
	if fileExists(filepath.Join(root, "cannon.previous.latest")) {
		t.Fatal("must not stage as cannon.previous.latest")
	}
	staged, err := os.ReadFile(filepath.Join(root, "cannon.latest"))
	if err != nil {
		t.Fatal(err)
	}
	if string(staged) != "staged-binary" {
		t.Fatalf("staged = %q", staged)
	}
}

func TestScheduleRestartCallsExitHook(t *testing.T) {
	done := make(chan struct{}, 1)
	prev := exitAfterRestart
	exitAfterRestart = func() { done <- struct{}{} }
	defer func() { exitAfterRestart = prev }()

	mgr := New(&config.App{DataRoot: t.TempDir()})
	mgr.ScheduleRestart()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("expected exit hook")
	}
}
