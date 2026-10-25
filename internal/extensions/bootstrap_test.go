package extensions

import (
	"context"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/rob121/cannon/internal/config"
	"github.com/rob121/cannon/internal/database"
	"github.com/rob121/cannon/internal/models"
	"github.com/rob121/cannon/internal/sites"
)

func TestSyncDirectoryReconcilesStaleSocketPath(t *testing.T) {
	root := t.TempDir()
	extDir := filepath.Join(root, "extensions")
	sockDir := filepath.Join(root, "sockets")
	if err := os.MkdirAll(extDir, 0755); err != nil {
		t.Fatal(err)
	}
	bin := filepath.Join(extDir, "cannon-ext-demo")
	if err := os.WriteFile(bin, []byte("#!/bin/sh\n"), 0755); err != nil {
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
	app.Extensions.SocketsDir = sockDir
	mgr := NewManager(app, site)

	stale := "/Users/someone/tmp/cannon/data/sockets/deadbeef.sock"
	if err := db.Create(&models.Extension{
		Name:   "cannon-ext-demo",
		Socket: stale,
		Status: models.StatusInactive,
	}).Error; err != nil {
		t.Fatal(err)
	}

	if err := mgr.syncDirectory(db); err != nil {
		t.Fatal(err)
	}
	var saved models.Extension
	if err := db.Where("name = ?", "cannon-ext-demo").First(&saved).Error; err != nil {
		t.Fatal(err)
	}
	want := mgr.socketPath("cannon-ext-demo")
	if saved.Socket != want {
		t.Fatalf("socket = %q, want %q (stale path was not reconciled)", saved.Socket, want)
	}
}

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

func TestRunParallelLimitsInFlightWork(t *testing.T) {
	var current, maxInFlight atomic.Int32
	runParallel(extensionStartParallelism, 12, func(i int) {
		n := current.Add(1)
		for {
			old := maxInFlight.Load()
			if n <= old || maxInFlight.CompareAndSwap(old, n) {
				break
			}
		}
		time.Sleep(30 * time.Millisecond)
		current.Add(-1)
		_ = i
	})
	if got := maxInFlight.Load(); got != extensionStartParallelism {
		t.Fatalf("max in flight = %d, want %d", got, extensionStartParallelism)
	}
}

func TestBootstrapStartsActiveExtensionsTogether(t *testing.T) {
	root := t.TempDir()
	extDir := filepath.Join(root, "extensions")
	if err := os.MkdirAll(extDir, 0755); err != nil {
		t.Fatal(err)
	}
	script := "#!/bin/sh\nfor arg in \"$@\"; do\n  case \"$arg\" in\n    --data=all) echo extensionname=demo; exit 0 ;;\n  esac\ndone\nexec sleep 30\n"
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
	names := []string{"cannon-ext-a", "cannon-ext-b", "cannon-ext-c"}
	for _, name := range names {
		path := filepath.Join(extDir, name)
		if err := os.WriteFile(path, []byte(script), 0755); err != nil {
			t.Fatal(err)
		}
		if err := db.Create(&models.Extension{Name: name, Socket: mgr.socketPath(name), Status: models.StatusActive}).Error; err != nil {
			t.Fatal(err)
		}
	}
	ctx := sites.WithContext(context.Background(), site)
	start := time.Now()
	if err := mgr.Bootstrap(ctx); err != nil {
		t.Fatal(err)
	}
	elapsed := time.Since(start)
	// Each missing socket waits about two seconds. Sequential startup would
	// take at least six seconds; a batch of five finishes near one wait.
	if elapsed >= 5*time.Second {
		t.Fatalf("bootstrap took %s, want extensions to start together", elapsed)
	}
	for _, name := range names {
		if !mgr.isSuppressed(name) {
			t.Fatalf("%s should be suppressed after the socket never appeared", name)
		}
		if mgr.IsRunning(name) {
			t.Fatalf("%s should not stay running", name)
		}
	}
}
