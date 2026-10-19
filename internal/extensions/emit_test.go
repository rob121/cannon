package extensions

import (
	"context"
	"net/http"
	"sync/atomic"
	"testing"

	"github.com/rob121/cannon/extension"
	"github.com/rob121/cannon/internal/hooks"
)

func TestFireEmitEventsAllowlist(t *testing.T) {
	hooks.Clear()
	t.Cleanup(hooks.Clear)
	var allowedHits atomic.Int32
	var blockedHits atomic.Int32
	hooks.Register(hooks.OnSubmissionAfterCreate, func(ctx context.Context, e *hooks.Event) (*hooks.Result, error) {
		allowedHits.Add(1)
		return nil, nil
	})
	hooks.Register(hooks.OnUserBeforeLogin, func(ctx context.Context, e *hooks.Event) (*hooks.Result, error) {
		blockedHits.Add(1)
		return nil, nil
	})

	FireEmitEvents(context.Background(), WireResponse{
		StatusCode: http.StatusSeeOther,
		EmitEvents: []extension.EmitEvent{
			{Event: hooks.OnSubmissionAfterCreate, Arguments: map[string]any{"title": "Tip"}},
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
	var hits atomic.Int32
	hooks.Register(hooks.OnSubmissionAfterCreate, func(ctx context.Context, e *hooks.Event) (*hooks.Result, error) {
		hits.Add(1)
		return nil, nil
	})
	FireEmitEvents(context.Background(), WireResponse{
		StatusCode: http.StatusBadRequest,
		EmitEvents: []extension.EmitEvent{
			{Event: hooks.OnSubmissionAfterCreate},
		},
	})
	if hits.Load() != 0 {
		t.Fatalf("hits = %d", hits.Load())
	}
}
