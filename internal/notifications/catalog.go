package notifications

import (
	"context"
	"encoding/json"
	"sort"
	"strings"

	"github.com/rob121/cannon/extension"
	"github.com/rob121/cannon/internal/hooks"
	"github.com/rob121/cannon/internal/models"
	"github.com/rob121/cannon/internal/sites"
	"gorm.io/gorm"
)

// EventInfo is a catalog entry for admin / profile UIs.
type EventInfo struct {
	Event        string
	Label        string
	Description  string
	Placeholders []extension.NotificationPlaceholder
	Extension    string
}

// CoreSubscribableEvents are built-in Layer 2 events (not extension-owned).
var CoreSubscribableEvents = []string{
	hooks.OnUserSignup,
	hooks.OnUserAfterLogin,
	hooks.OnUserVerified,
	hooks.OnUserLocked,
	hooks.OnUserLogout,
	hooks.OnItemAfterSave,
	hooks.OnCommentAfterSave,
}

// CoreNotificationEvents are built-in Layer 1 destination events.
var CoreNotificationEvents = []string{
	hooks.OnUserSignup,
	hooks.OnUserAfterLogin,
	hooks.OnUserVerified,
	hooks.OnUserLocked,
	hooks.OnItemAfterSave,
	hooks.OnCommentAfterSave,
}

// SubscribableEvents is the core list kept for init-time hook registration.
// Prefer SubscribableEventsFor(ctx) in request handlers.
var SubscribableEvents = CoreSubscribableEvents

// NotificationEvents is the core Layer 1 list; prefer NotificationEventsFor(ctx).
var NotificationEvents = CoreNotificationEvents

func loadExtensionEventRows(ctx context.Context) ([]models.ExtensionNotificationEvent, error) {
	db, err := sites.DB(ctx)
	if err != nil {
		return nil, err
	}
	var rows []models.ExtensionNotificationEvent
	if err := db.Order("group_id ASC, event ASC").Find(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}

func parsePlaceholders(raw string) []extension.NotificationPlaceholder {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	var out []extension.NotificationPlaceholder
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		return nil
	}
	return out
}

// ExtensionEventInfos returns persisted extension notification event definitions.
func ExtensionEventInfos(ctx context.Context) ([]EventInfo, error) {
	rows, err := loadExtensionEventRows(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]EventInfo, 0, len(rows))
	for _, row := range rows {
		out = append(out, EventInfo{
			Event:        row.Event,
			Label:        row.Label,
			Description:  row.Description,
			Placeholders: parsePlaceholders(row.PlaceholdersJSON),
			Extension:    row.ExtensionName,
		})
	}
	return out, nil
}

// CanEmit reports whether an extension may fire this event via emit_events.
func CanEmit(ctx context.Context, event string) bool {
	event = strings.TrimSpace(event)
	if event == "" {
		return false
	}
	if hooks.IsEmittable(event) {
		return true
	}
	db, err := sites.DB(ctx)
	if err != nil {
		return false
	}
	var count int64
	if err := db.Model(&models.ExtensionNotificationEvent{}).Where("event = ?", event).Count(&count).Error; err != nil {
		return false
	}
	return count > 0
}

// NotificationEventsFor returns Layer 1 events (core + extensions).
func NotificationEventsFor(ctx context.Context) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(CoreNotificationEvents)+4)
	for _, event := range CoreNotificationEvents {
		seen[event] = struct{}{}
		out = append(out, event)
	}
	rows, err := loadExtensionEventRows(ctx)
	if err != nil {
		return out
	}
	for _, row := range rows {
		if _, ok := seen[row.Event]; ok {
			continue
		}
		seen[row.Event] = struct{}{}
		out = append(out, row.Event)
	}
	return out
}

// SubscribableEventsFor returns Layer 2 events (core + extensions).
func SubscribableEventsFor(ctx context.Context) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(CoreSubscribableEvents)+4)
	for _, event := range CoreSubscribableEvents {
		seen[event] = struct{}{}
		out = append(out, event)
	}
	rows, err := loadExtensionEventRows(ctx)
	if err != nil {
		return out
	}
	for _, row := range rows {
		if _, ok := seen[row.Event]; ok {
			continue
		}
		seen[row.Event] = struct{}{}
		out = append(out, row.Event)
	}
	return out
}

// EventGroupsFor returns grouped events for checklists (core + extensions).
func EventGroupsFor(ctx context.Context) []EventGroup {
	groups := []EventGroup{
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
	rows, err := loadExtensionEventRows(ctx)
	if err != nil || len(rows) == 0 {
		return groups
	}
	byGroup := map[string]*EventGroup{}
	order := make([]string, 0)
	for _, row := range rows {
		groupID := strings.TrimSpace(row.GroupID)
		if groupID == "" {
			groupID = "extensions"
		}
		g, ok := byGroup[groupID]
		if !ok {
			label := strings.TrimSpace(row.GroupLabel)
			if label == "" {
				label = "Extensions"
			}
			g = &EventGroup{ID: groupID, Label: label}
			byGroup[groupID] = g
			order = append(order, groupID)
		}
		g.Events = append(g.Events, row.Event)
	}
	sort.Strings(order)
	for _, id := range order {
		g := byGroup[id]
		sort.Strings(g.Events)
		groups = append(groups, *g)
	}
	return groups
}

// EventGroups returns core-only groups (no DB). Prefer EventGroupsFor(ctx).
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

func isSubscribableEvent(ctx context.Context, event string) bool {
	event = strings.TrimSpace(event)
	if event == "" {
		return false
	}
	for _, item := range SubscribableEventsFor(ctx) {
		if item == event {
			return true
		}
	}
	return false
}

func isKnownSubscriptionEvent(db *gorm.DB, event string) bool {
	event = strings.TrimSpace(event)
	if event == "" {
		return false
	}
	for _, item := range CoreSubscribableEvents {
		if item == event {
			return true
		}
	}
	var count int64
	if err := db.Model(&models.ExtensionNotificationEvent{}).Where("event = ?", event).Count(&count).Error; err != nil {
		return false
	}
	return count > 0
}

// PlaceholdersForEvent returns template placeholders for an event when known.
func PlaceholdersForEvent(ctx context.Context, event string) []extension.NotificationPlaceholder {
	event = strings.TrimSpace(event)
	rows, err := loadExtensionEventRows(ctx)
	if err != nil {
		return nil
	}
	for _, row := range rows {
		if row.Event == event {
			return parsePlaceholders(row.PlaceholdersJSON)
		}
	}
	return nil
}
