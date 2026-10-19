package extensions

import (
	"context"
	"net/http"
	"sync/atomic"
	"testing"

	"github.com/rob121/cannon/extension"
	"github.com/rob121/cannon/internal/config"
	"github.com/rob121/cannon/internal/database"
	"github.com/rob121/cannon/internal/hooks"
	"github.com/rob121/cannon/internal/models"
	"github.com/rob121/cannon/internal/sites"
)

func emitTestContext(t *testing.T) context.Context {
	t.Helper()
	path := t.TempDir() + "/emit.sqlite"
	site := &config.SiteConfig{
		ID:       t.Name(),
		Database: config.DatabaseConfig{Type: "sqlite", DSN: path},
	}
	if err := database.Migrate(site); err != nil {
		t.Fatal(err)
	}
	db, err := database.Get(site.ID)
	if err != nil {
		t.Fatal(err)
	}
	ctx := sites.WithContext(context.Background(), site)
	row := models.ExtensionNotificationEvent{
		ExtensionName: "cannon-ext-test",
		Event:         "onTestEmit",
		Label:         "Test emit",
		GroupID:       "tests",
		GroupLabel:    "Tests",
	}
	if err := db.Create(&row).Error; err != nil {
		t.Fatal(err)
	}
	return ctx
}

func TestFireEmitEventsAllowlist(t *testing.T) {
	hooks.Clear()
	t.Cleanup(hooks.Clear)
	ctx := emitTestContext(t)

	var allowedHits atomic.Int32
	var blockedHits atomic.Int32
	hooks.Register("onTestEmit", func(ctx context.Context, e *hooks.Event) (*hooks.Result, error) {
		allowedHits.Add(1)
		return nil, nil
	})
	hooks.Register(hooks.OnUserBeforeLogin, func(ctx context.Context, e *hooks.Event) (*hooks.Result, error) {
		blockedHits.Add(1)
		return nil, nil
	})

	FireEmitEvents(ctx, WireResponse{
		StatusCode: http.StatusSeeOther,
		EmitEvents: []extension.EmitEvent{
			{Event: "onTestEmit", Arguments: map[string]any{"title": "Tip"}},
			{Event: hooks.OnUserBeforeLogin, Arguments: map[string]any{"username": "nope"}},
		},
	})
	if allowedHits.Load() != 1 {
		t.Fatalf("allowed hits = %d", allowedHits.Load())
	}
	if blockedHits.Load() != 0 {
		t.Fatalf("blocked hits = %d", blockedHits.Load())
	}
}

func TestFireEmitEventsSkipsErrorStatus(t *testing.T) {
	hooks.Clear()
	t.Cleanup(hooks.Clear)
	ctx := emitTestContext(t)

	var hits atomic.Int32
	hooks.Register("onTestEmit", func(ctx context.Context, e *hooks.Event) (*hooks.Result, error) {
		hits.Add(1)
		return nil, nil
	})
	FireEmitEvents(ctx, WireResponse{
		StatusCode: http.StatusBadRequest,
		EmitEvents: []extension.EmitEvent{
			{Event: "onTestEmit"},
		},
	})
	if hits.Load() != 0 {
		t.Fatalf("hits = %d", hits.Load())
	}
}
