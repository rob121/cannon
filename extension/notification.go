package extension

// NotificationPlaceholder describes one value an emit payload may include.
type NotificationPlaceholder struct {
	Key         string `json:"key"`
	Label       string `json:"label,omitempty"`
	Description string `json:"description,omitempty"`
}

// NotificationEventDefinition describes a host notification event this extension can emit.
type NotificationEventDefinition struct {
	// Event is the hook name Cannon will accept in WireResponse.emit_events
	// (for example "onSubmissionAfterCreate"). Must be unique across the site.
	Event string `json:"event"`
	// Label is a short UI title; defaults to Event when empty.
	Label string `json:"label,omitempty"`
	// Group and GroupLabel organize the event in admin / profile checklists.
	Group      string `json:"group,omitempty"`
	GroupLabel string `json:"group_label,omitempty"`
	// Description explains when the event fires.
	Description string `json:"description,omitempty"`
	// Placeholders document emit_events argument keys for message templates.
	Placeholders []NotificationPlaceholder `json:"placeholders,omitempty"`
}

// NotificationEventListResponse is returned from GET /notifications.
type NotificationEventListResponse struct {
	Events []NotificationEventDefinition `json:"events"`
}
