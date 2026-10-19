package notifications

// EventGroup organizes subscribable hooks for admin and account UIs.
type EventGroup struct {
	ID     string
	Label  string
	Events []string
}
