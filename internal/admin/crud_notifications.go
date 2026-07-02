package admin

import (
	"net/http"
	"strings"

	"github.com/rob121/cannon/internal/models"
	"github.com/rob121/cannon/internal/notifications"
	"github.com/rob121/cannon/internal/sites"
	"gorm.io/gorm"
)

const notificationsBase = "/admin/notifications"

func (h *Handler) notifications(w http.ResponseWriter, r *http.Request, path string) {
	parts := pathParts("/notifications", path)
	switch {
	case len(parts) == 0:
		h.notificationList(w, r)
	case parts[0] == "new":
		h.notificationForm(w, r, 0)
	case len(parts) == 2 && parts[1] == "delete":
		h.notificationDelete(w, r, parts[0])
	case len(parts) == 2 && parts[1] == "toggle-status":
		h.notificationToggleStatus(w, r, parts[0])
	default:
		id, ok := parseID(parts[0])
		if !ok {
			http.NotFound(w, r)
			return
		}
		h.notificationForm(w, r, id)
	}
}

func (h *Handler) notificationList(w http.ResponseWriter, r *http.Request) {
	db, _ := sites.DB(r.Context())
	page := parsePage(r)
	var total int64
	db.Model(&models.Notification{}).Count(&total)
	data := listPage(page, total, notificationsBase,
		"Send alerts via shoutrrr when user lifecycle hooks fire.",
		"Add Notification", map[string]any{"ActiveNav": "notifications"})
	order := applyListSort(r, data, map[string]string{
		"name": "name", "status": "status",
	}, "name")
	var rows []models.Notification
	db.Preload("Events").Offset((page - 1) * pageSize).Limit(pageSize).Order(order).Find(&rows)
	data["Rows"] = rows
	h.render(w, r, "Notifications", "admin/notifications.html", data)
}

func (h *Handler) notificationForm(w http.ResponseWriter, r *http.Request, id uint) {
	db, _ := sites.DB(r.Context())
	isNew := id == 0
	var row models.Notification
	if !isNew {
		if err := db.Preload("Events").First(&row, id).Error; err != nil {
			http.NotFound(w, r)
			return
		}
	} else {
		row.Status = models.StatusActive
	}
	selectedEvents := notificationEventSet(row.Events)

	if r.Method == http.MethodPost {
		if err := r.ParseForm(); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		row.Name = formString(r, "name")
		row.ShoutrrURL = formString(r, "shoutrr_url")
		row.Status = formStatus(r)
		selectedEvents = r.Form["events"]
		if row.Name == "" || row.ShoutrrURL == "" {
			h.renderNotificationForm(w, r, row, selectedEvents, isNew, "Name and shoutrrr URL are required.")
			return
		}
		var saveErr error
		if isNew {
			saveErr = db.Create(&row).Error
		} else {
			saveErr = db.Save(&row).Error
		}
		if saveErr != nil {
			h.renderNotificationForm(w, r, row, selectedEvents, isNew, saveErr.Error())
			return
		}
		if err := replaceNotificationEvents(db, row.NotificationID, r.Form["events"]); err != nil {
			h.renderNotificationForm(w, r, row, selectedEvents, isNew, err.Error())
			return
		}
		redirectList(w, r, notificationsBase)
		return
	}
	h.renderNotificationForm(w, r, row, selectedEvents, isNew, "")
}

func replaceNotificationEvents(db *gorm.DB, notificationID uint, events []string) error {
	if err := db.Where("notification_id = ?", notificationID).Delete(&models.NotificationEvent{}).Error; err != nil {
		return err
	}
	for _, event := range events {
		event = strings.TrimSpace(event)
		if event == "" {
			continue
		}
		if err := db.Create(&models.NotificationEvent{NotificationID: notificationID, Event: event}).Error; err != nil {
			return err
		}
	}
	return nil
}

func notificationEventSet(events []models.NotificationEvent) []string {
	out := make([]string, 0, len(events))
	for _, e := range events {
		out = append(out, e.Event)
	}
	return out
}

func (h *Handler) renderNotificationForm(w http.ResponseWriter, r *http.Request, row models.Notification, selected []string, isNew bool, errMsg string) {
	title := "Add Notification"
	if !isNew {
		title = "Edit Notification"
	}
	data := formData(map[string]any{
		"ActiveNav":      "notifications",
		"Row":            row,
		"IsNew":          isNew,
		"BasePath":       notificationsBase,
		"SelectedEvents": selected,
		"HookEvents":     notifications.NotificationEvents,
	})
	if errMsg != "" {
		data["Error"] = errMsg
	}
	h.render(w, r, title, "admin/notifications_form.html", data)
}

func (h *Handler) notificationDelete(w http.ResponseWriter, r *http.Request, idStr string) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	id, ok := parseID(idStr)
	if !ok {
		http.NotFound(w, r)
		return
	}
	db, _ := sites.DB(r.Context())
	db.Where("notification_id = ?", id).Delete(&models.NotificationEvent{})
	if err := db.Delete(&models.Notification{}, id).Error; err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	redirectList(w, r, notificationsBase)
}

func (h *Handler) notificationToggleStatus(w http.ResponseWriter, r *http.Request, idStr string) {
	h.postToggleModel(w, r, idStr, &models.Notification{}, notificationsBase)
}
