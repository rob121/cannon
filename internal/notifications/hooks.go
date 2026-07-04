package notifications

import (
	"context"

	"github.com/rob121/cannon/internal/hooks"
)

func init() {
	seen := map[string]struct{}{}
	for _, event := range SubscribableEvents {
		if _, ok := seen[event]; ok {
			continue
		}
		seen[event] = struct{}{}
		event := event
		hooks.Register(event, func(ctx context.Context, e *hooks.Event) (*hooks.Result, error) {
			DispatchEvent(ctx, event, e.Arguments)
			return nil, nil
		})
	}
}
