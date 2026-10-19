package extensions

import (
	"context"
	"log"
	"net/http"
	"strings"

	"github.com/rob121/cannon/internal/hooks"
	"github.com/rob121/cannon/internal/notifications"
)

// FireEmitEvents dispatches allowlisted notification events requested by an extension wire response.
// Events must be registered via the extension notifications capability (or rare core EmittableEvents).
// Failures are logged and never returned to the browser.
func FireEmitEvents(ctx context.Context, wire WireResponse) {
	if len(wire.EmitEvents) == 0 {
		return
	}
	if wire.StatusCode >= http.StatusBadRequest {
		return
	}
	for _, item := range wire.EmitEvents {
		event := strings.TrimSpace(item.Event)
		if event == "" {
			continue
		}
		if !notifications.CanEmit(ctx, event) {
			log.Printf("extensions: ignore unregistered emit event %q", event)
			continue
		}
		args := item.Arguments
		if args == nil {
			args = map[string]any{}
		}
		notifications.DispatchEvent(ctx, event, args)
		if _, err := hooks.Fire(ctx, event, args); err != nil {
			log.Printf("extensions: emit hook %q: %v", event, err)
		}
	}
}
