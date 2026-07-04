package notifications

import "github.com/rob121/cannon/internal/hooks"

// EventGroup organizes subscribable hooks for admin and account UIs.
type EventGroup struct {
	ID     string
	Label  string
	Events []string
}

// SubscribableEvents lists hooks available for Layer 2 user/role subscriptions.
var SubscribableEvents = []string{
	hooks.OnUserSignup,
	hooks.OnUserAfterLogin,
	hooks.OnUserVerified,
	hooks.OnUserLocked,
	hooks.OnUserLogout,
	hooks.OnItemAfterSave,
	hooks.OnCommentAfterSave,
}

// NotificationEvents lists hooks available for Layer 1 admin destinations.
var NotificationEvents = []string{
	hooks.OnUserSignup,
	hooks.OnUserAfterLogin,
	hooks.OnUserVerified,
	hooks.OnUserLocked,
	hooks.OnItemAfterSave,
	hooks.OnCommentAfterSave,
}

// EventGroups returns subscribable events grouped for display.
func EventGroups() []EventGroup {
	return []EventGroup{
		{
			ID:    "account",
			Label: "Account",
			Events: []string{
				hooks.OnUserSignup,
				hooks.OnUserAfterLogin,
				hooks.OnUserVerified,
				hooks.OnUserLocked,
				hooks.OnUserLogout,
			},
		},
		{
			ID:     "content",
			Label:  "Content",
			Events: []string{hooks.OnItemAfterSave},
		},
		{
			ID:     "comments",
			Label:  "Comments",
			Events: []string{hooks.OnCommentAfterSave},
		},
	}
}

func isSubscribableEvent(event string) bool {
	for _, item := range SubscribableEvents {
		if item == event {
			return true
		}
	}
	return false
}
