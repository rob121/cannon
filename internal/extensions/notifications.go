package extensions

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/rob121/cannon/extension"
	"github.com/rob121/cannon/internal/models"
	"github.com/rob121/cannon/internal/sites"
	"gorm.io/gorm"
)

type notificationEventListResponse struct {
	Events []extension.NotificationEventDefinition `json:"events"`
}

func (m *Manager) fetchNotificationEvents(socketPath, path string) ([]extension.NotificationEventDefinition, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		path = "/notifications"
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	path = strings.TrimRight(path, "/")
	resp, err := m.do(socketPath, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var payload notificationEventListResponse
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, err
	}
	return payload.Events, nil
}

func (m *Manager) clearInactiveNotificationEvents(_ context.Context, db *gorm.DB) error {
	if db == nil || !db.Migrator().HasTable(&models.ExtensionNotificationEvent{}) {
		return nil
	}
	var active []string
	if err := db.Model(&models.Extension{}).Where("status = ?", models.StatusActive).Pluck("name", &active).Error; err != nil {
		return err
	}
	q := db.Where("1 = 1")
	if len(active) > 0 {
		q = db.Where("extension_name NOT IN ?", active)
	}
	return q.Delete(&models.ExtensionNotificationEvent{}).Error
}

func (m *Manager) syncNotificationEvents(ctx context.Context, extensionName string, events []extension.NotificationEventDefinition) error {
	extensionName = strings.TrimSpace(extensionName)
	if extensionName == "" {
		return nil
	}
	db, err := sites.DB(ctx)
	if err != nil {
		return err
	}
	if !db.Migrator().HasTable(&models.ExtensionNotificationEvent{}) {
		return nil
	}
	if err := db.Where("extension_name = ?", extensionName).Delete(&models.ExtensionNotificationEvent{}).Error; err != nil {
		return err
	}
	for _, def := range events {
		event := strings.TrimSpace(def.Event)
		if event == "" {
			continue
		}
		label := strings.TrimSpace(def.Label)
		if label == "" {
			label = event
		}
		groupID := strings.TrimSpace(def.Group)
		if groupID == "" {
			groupID = "extensions"
		}
		groupLabel := strings.TrimSpace(def.GroupLabel)
		if groupLabel == "" {
			groupLabel = "Extensions"
		}
		placeholdersJSON := ""
		if len(def.Placeholders) > 0 {
			raw, err := json.Marshal(def.Placeholders)
			if err == nil {
				placeholdersJSON = string(raw)
			}
		}
		row := models.ExtensionNotificationEvent{
			ExtensionName:    extensionName,
			Event:            event,
			Label:            label,
			GroupID:          groupID,
			GroupLabel:       groupLabel,
			Description:      strings.TrimSpace(def.Description),
			PlaceholdersJSON: placeholdersJSON,
		}
		if err := db.Create(&row).Error; err != nil {
			return err
		}
	}
	return nil
}
