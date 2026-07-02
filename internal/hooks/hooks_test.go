package hooks

import (
	"context"
	"errors"
	"testing"
)

func TestFireLocalMergeAndStop(t *testing.T) {
	Clear()
	t.Cleanup(Clear)

	Register(OnBeforeRoute, func(_ context.Context, e *Event) (*Result, error) {
		return &Result{Arguments: map[string]any{"seen": true}}, nil
	})
	Register(OnBeforeRoute, func(_ context.Context, e *Event) (*Result, error) {
		return &Result{Stop: true}, nil
	})
	Register(OnBeforeRoute, func(_ context.Context, e *Event) (*Result, error) {
		t.Fatal("should not run after stop")
		return nil, nil
	})

	out, stop, err := FireLocal(context.Background(), nil, OnBeforeRoute, map[string]any{})
	if err != nil || !stop || out["seen"] != true {
		t.Fatalf("out=%v stop=%v err=%v", out, stop, err)
	}
}

func TestFireLocalLoginBlocked(t *testing.T) {
	Clear()
	t.Cleanup(Clear)

	Register(OnUserBeforeLogin, func(_ context.Context, e *Event) (*Result, error) {
		return &Result{Arguments: map[string]any{"allowed": false, "error": "nope"}}, nil
	})
	_, _, err := FireLocal(context.Background(), nil, OnUserBeforeLogin, map[string]any{})
	if !errors.Is(err, ErrAborted) {
		t.Fatalf("expected ErrAborted, got %v", err)
	}
}

func TestWrapAbort(t *testing.T) {
	if !errors.Is(WrapAbort("blocked"), ErrAborted) {
		t.Fatal("expected wrapped abort")
	}
}
