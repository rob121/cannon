package realtime

import (
	"context"
	"testing"

	"github.com/rob121/cannon/internal/config"
	"github.com/rob121/cannon/internal/hooks"
	"github.com/rob121/cannon/internal/sites"
)

func TestBodyFragmentDisabledByDefault(t *testing.T) {
	hub, err := NewHub()
	if err != nil {
		t.Fatal(err)
	}
	defer hub.Shutdown(context.Background())
	SetHub(hub)
	defer SetHub(nil)

	ctx := sites.WithContext(context.Background(), &config.SiteConfig{ID: "demo", Host: "example.test"})
	frag, ok := BodyFragment(ctx, nil)
	if ok || frag != "" {
		t.Fatalf("expected disabled fragment, got ok=%v frag=%q", ok, frag)
	}
}

func TestAnalyticsDocumentBodyHookSkipsAdmin(t *testing.T) {
	out, err := analyticsDocumentBodyHook(context.Background(), &hooks.Event{
		Arguments: map[string]any{"context": "admin"},
	})
	if err != nil || out != nil {
		t.Fatalf("expected nil result for admin, got %#v err=%v", out, err)
	}
}
