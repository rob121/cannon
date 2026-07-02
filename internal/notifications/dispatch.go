package notifications

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/rob121/cannon/internal/models"
	"github.com/rob121/cannon/internal/sites"
)

// Send delivers a message via a shoutrrr service URL.
func Send(ctx context.Context, shoutrrURL, message string) error {
	shoutrrURL = strings.TrimSpace(shoutrrURL)
	if shoutrrURL == "" {
		return fmt.Errorf("shoutrrr url is required")
	}
	return sendShoutrr(ctx, shoutrrURL, message)
}

// DispatchEvent sends notifications configured for the given hook event.
func DispatchEvent(ctx context.Context, event string, args map[string]any) {
	db, err := sites.DB(ctx)
	if err != nil {
		return
	}
	event = strings.TrimSpace(event)
	if event == "" {
		return
	}
	var rows []models.Notification
	if err := db.Preload("Events").
		Joins("JOIN notification_events ON notification_events.notification_id = notifications.notification_id").
		Where("notification_events.event = ? AND notifications.status = ?", event, models.StatusActive).
		Find(&rows).Error; err != nil {
		log.Printf("notifications: load for %s: %v", event, err)
		return
	}
	if len(rows) == 0 {
		return
	}
	message := formatMessage(event, args)
	for _, row := range rows {
		if err := Send(ctx, row.ShoutrrURL, message); err != nil {
			log.Printf("notifications: send %q (%s): %v", row.Name, event, err)
		}
	}
}

func formatMessage(event string, args map[string]any) string {
	parts := []string{"Cannon: " + event}
	for _, key := range []string{"username", "email", "user_id"} {
		if v, ok := args[key]; ok && fmt.Sprint(v) != "" {
			parts = append(parts, fmt.Sprintf("%s=%v", key, v))
		}
	}
	return strings.Join(parts, " ")
}
