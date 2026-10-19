package extensions

import (
	"context"
	"log"
	"net/http"
	"strings"

	"github.com/rob121/cannon/internal/hooks"
)

// FireEmitEvents fires allowlisted host hooks requested by an extension wire response.
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
		if !hooks.IsEmittable(event) {
			log.Printf("extensions: ignore non-emittable event %q", event)
			continue
		}
		args := item.Arguments
		if args == nil {
			args = map[string]any{}
		}
		if _, err := hooks.Fire(ctx, event, args); err != nil {
			log.Printf("extensions: emit %s: %v", event, err)
		}
	}
}
