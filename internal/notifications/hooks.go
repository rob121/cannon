package notifications

import (
	"context"

	"github.com/rob121/cannon/internal/hooks"
)

// NotificationEvents lists hook names available for notification subscriptions.
var NotificationEvents = []string{
	hooks.OnUserSignup,
	hooks.OnUserAfterLogin,
	hooks.OnUserVerified,
	hooks.OnUserLocked,
}

func init() {
	for _, event := range NotificationEvents {
		event := event
		hooks.Register(event, func(ctx context.Context, e *hooks.Event) (*hooks.Result, error) {
			DispatchEvent(ctx, event, e.Arguments)
			return nil, nil
		})
	}
}
