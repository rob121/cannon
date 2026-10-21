package notifications

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/rob121/cannon/internal/models"
	"github.com/rob121/cannon/internal/sites"
	"gorm.io/gorm"
)

// Send delivers a message via a shoutrrr service URL.
func Send(ctx context.Context, shoutrrURL, message string) error {
	return SendWithSubject(ctx, shoutrrURL, "", message)
}

// SendWithSubject delivers a message, optionally overriding the SMTP subject.
func SendWithSubject(ctx context.Context, shoutrrURL, subject, message string) error {
	shoutrrURL = strings.TrimSpace(shoutrrURL)
	if shoutrrURL == "" {
		return fmt.Errorf("shoutrrr url is required")
	}
	return sendShoutrr(ctx, shoutrrURL, subject, message)
}

// SendTest delivers a sample message for one saved notification destination.
// Success and failure update the row's last-sent / last-error fields.
func SendTest(ctx context.Context, notificationID uint) error {
	db, err := sites.DB(ctx)
	if err != nil {
		return err
	}
	var row models.Notification
	if err := db.First(&row, notificationID).Error; err != nil {
		return err
	}
	name := strings.TrimSpace(row.Name)
	if name == "" {
		name = fmt.Sprintf("#%d", row.NotificationID)
	}
	subject := "Cannon notification test"
	body := fmt.Sprintf("This is a test message from Cannon for notification %q.", name)
	if err := SendWithSubject(ctx, row.ShoutrrURL, subject, body); err != nil {
		_ = recordDeliveryFailure(db, row.NotificationID, err.Error())
		return err
	}
	return recordDeliverySuccess(db, row.NotificationID)
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
	rows, err := notificationsForEvent(ctx, event)
	if err != nil {
		log.Printf("notifications: load for %s: %v", event, err)
	} else if len(rows) > 0 {
		for _, row := range rows {
			subject, body := RenderEventMessage(row.TemplatesJSON, event, args)
			if err := SendWithSubject(ctx, row.ShoutrrURL, subject, body); err != nil {
				log.Printf("notifications: send %q (%s): %v", row.Name, event, err)
				_ = recordDeliveryFailure(db, row.NotificationID, err.Error())
				continue
			}
			_ = recordDeliverySuccess(db, row.NotificationID)
		}
	}
	dispatchLayer2(ctx, event, args)
}

// notificationsForEvent returns active notifications subscribed to event.
// Rows owned by a user are included only when that user can access the admin.
// Rows with no user still send so existing destinations keep working until they are assigned.
func notificationsForEvent(ctx context.Context, event string) ([]models.Notification, error) {
	db, err := sites.DB(ctx)
	if err != nil {
		return nil, err
	}
	var rows []models.Notification
	if err := db.Preload("Events").
		Joins("JOIN notification_events ON notification_events.notification_id = notifications.notification_id").
		Where("notification_events.event = ? AND notifications.status = ?", event, models.StatusActive).
		Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]models.Notification, 0, len(rows))
	for _, row := range rows {
		if row.UserID == 0 {
			out = append(out, row)
			continue
		}
		admin, err := userHasAdminAccess(ctx, row.UserID)
		if err != nil {
			return nil, err
		}
		if !admin {
			continue
		}
		out = append(out, row)
	}
	return out, nil
}

func recordDeliverySuccess(db *gorm.DB, notificationID uint) error {
	now := time.Now().UTC()
	return db.Model(&models.Notification{}).Where("notification_id = ?", notificationID).Updates(map[string]any{
		"last_error":    "",
		"last_error_at": nil,
		"last_sent_at":  now,
	}).Error
}

func recordDeliveryFailure(db *gorm.DB, notificationID uint, errMsg string) error {
	now := time.Now().UTC()
	errMsg = strings.TrimSpace(errMsg)
	if len(errMsg) > 2000 {
		errMsg = errMsg[:2000]
	}
	return db.Model(&models.Notification{}).Where("notification_id = ?", notificationID).Updates(map[string]any{
		"last_error":    errMsg,
		"last_error_at": now,
	}).Error
}
