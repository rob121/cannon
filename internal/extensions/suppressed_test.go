package extensions

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/rob121/cannon/internal/config"
	"github.com/rob121/cannon/internal/models"
)

func TestStopSuppressesBootstrapAutoStart(t *testing.T) {
	m := NewManager(nil, nil)
	m.Stop("example")

	if !m.isSuppressed("example") {
		t.Fatal("expected manual stop to suppress auto-start")
	}
}

func TestStartReSuppressesOnFailure(t *testing.T) {
	root := t.TempDir()
	m := NewManager(&config.App{
		Extensions: config.ExtensionsConfig{Dir: root, SocketsDir: root},
	}, &config.SiteConfig{ID: "site"})
	m.suppressed["missing"] = true

	row := models.Extension{
		Name:   "missing",
		Socket: filepath.Join(root, "missing.sock"),
	}
	err := m.Start(context.Background(), row)
	if err == nil {
		t.Fatal("expected start to fail for missing binary")
	}
	if !m.isSuppressed("missing") {
		t.Fatal("expected failed start to suppress auto-start again")
	}
}
