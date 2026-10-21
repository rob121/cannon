package admin

import (
	"context"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/rob121/cannon/extension"
	"github.com/rob121/cannon/internal/mail"
	"github.com/rob121/cannon/internal/models"
	"github.com/rob121/cannon/internal/notifications"
	"github.com/rob121/cannon/internal/sites"
	"gorm.io/gorm"
)

type notificationListRow struct {
	models.Notification
	DeliveryLabel string
	DeliveryTime  string
	DeliveryError bool
	OwnerLabel    string
	EditBase      string
}

const notificationsBase = "/admin/notifications"

func userNotificationsBase(userID uint) string {
	return usersBase + "/" + strconv.FormatUint(uint64(userID), 10) + "/notifications"
}

func (h *Handler) notifications(w http.ResponseWriter, r *http.Request, path string) {
	h.notificationRoutes(w, r, nil, notificationsBase, pathParts("/notifications", path))
}

func (h *Handler) userNotifications(w http.ResponseWriter, r *http.Request, userID uint, parts []string) {
	db, err := sites.DB(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	var owner models.User
	if err := db.First(&owner, userID).Error; err != nil {
		h.notFound(w, r)
		return
	}
	h.notificationRoutes(w, r, &owner, userNotificationsBase(userID), parts)
}

func (h *Handler) notificationRoutes(w http.ResponseWriter, r *http.Request, owner *models.User, base string, parts []string) {
	switch {
	case len(parts) == 0:
		h.notificationList(w, r, owner, base)
	case parts[0] == "new":
		h.notificationForm(w, r, owner, base, 0)
	case len(parts) == 2 && parts[1] == "delete":
		h.notificationDelete(w, r, owner, base, parts[0])
	case len(parts) == 2 && parts[1] == "toggle-status":
		h.notificationToggleStatus(w, r, owner, base, parts[0])
	case len(parts) == 2 && parts[1] == "test":
		h.notificationTest(w, r, owner, base, parts[0])
	default:
		id, ok := parseID(parts[0])
		if !ok {
			h.notFound(w, r)
			return
		}
		h.notificationForm(w, r, owner, base, id)
	}
}

func (h *Handler) notificationList(w http.ResponseWriter, r *http.Request, owner *models.User, base string) {
	db, _ := sites.DB(r.Context())
	page := parsePage(r)
	countQ := db.Model(&models.Notification{})
	listQ := db.Model(&models.Notification{}).Preload("Events")
	if owner != nil {
		countQ = countQ.Where("user_id = ?", owner.UserID)
		listQ = listQ.Where("user_id = ?", owner.UserID)
	}
	var total int64
	countQ.Count(&total)
	activeNav := "notifications"
	subtitle := "Each notification belongs to an account and subscribes to hook events."
	addLabel := "Add Notification"
	if owner != nil {
		activeNav = "accounts"
		subtitle = "Notifications for " + notificationOwnerLabel(*owner) + ". Subscribed events are delivered to this notification."
	}
	data := listPage(r, page, total, base, subtitle, addLabel, map[string]any{"ActiveNav": activeNav})
	if owner != nil {
		data["AccountURL"] = usersBase + "/" + strconv.FormatUint(uint64(owner.UserID), 10)
		data["AccountLabel"] = notificationOwnerLabel(*owner)
	}
	order := applyListSort(r, data, map[string]string{
		"name": "name", "status": "status",
	}, "name")
	var rows []models.Notification
	listQ.Offset((page - 1) * pageSizeFor(r)).Limit(pageSizeFor(r)).Order(order).Find(&rows)
	owners := notificationOwnerNames(db, rows)
	listRows := make([]notificationListRow, 0, len(rows))
	for _, row := range rows {
		item := notificationListRow{Notification: row, OwnerLabel: owners[row.UserID]}
		if row.UserID > 0 {
			item.EditBase = userNotificationsBase(row.UserID)
		} else {
			item.EditBase = notificationsBase
			item.OwnerLabel = "Unassigned"
		}
		if strings.TrimSpace(row.LastError) != "" {
			item.DeliveryLabel = "Error"
			item.DeliveryError = true
			item.DeliveryTime = formatNotificationTime(row.LastErrorAt)
		} else if row.LastSentAt != nil {
			item.DeliveryLabel = "Sent"
			item.DeliveryTime = formatNotificationTime(row.LastSentAt)
		}
		listRows = append(listRows, item)
	}
	data["Rows"] = listRows
	if r.URL.Query().Get("tested") == "1" {
		data["Success"] = "Test message sent."
	}
	if r.URL.Query().Get("test_error") == "1" {
		data["Error"] = "Test message failed. Open the notification to see the delivery error."
	}
	h.render(w, r, "Notifications", "admin/notifications.html", data)
}

func formatNotificationTime(t *time.Time) string {
	if t == nil || t.IsZero() {
		return ""
	}
	return t.Local().Format("2006-01-02 15:04")
}

func (h *Handler) notificationForm(w http.ResponseWriter, r *http.Request, owner *models.User, base string, id uint) {
	db, _ := sites.DB(r.Context())
	isNew := id == 0
	var row models.Notification
	if !isNew {
		if err := db.Preload("Events").First(&row, id).Error; err != nil {
			h.notFound(w, r)
			return
		}
		if owner != nil && row.UserID != owner.UserID {
			h.notFound(w, r)
			return
		}
		if owner == nil && row.UserID > 0 && r.Method == http.MethodGet {
			http.Redirect(w, r, userNotificationsBase(row.UserID)+"/"+strconv.FormatUint(uint64(row.NotificationID), 10), http.StatusSeeOther)
			return
		}
	} else {
		row.Status = models.StatusActive
		if owner != nil {
			row.UserID = owner.UserID
		}
	}
	selectedEvents := notificationEventSet(row.Events)
	templates := notifications.ParseTemplatesJSON(row.TemplatesJSON)
	mailCfg, _ := mail.LoadSettings(r.Context())
	useSiteMail := false
	siteMailTo := ""
	if to, ok := mail.SiteMailRecipient(mailCfg, row.ShoutrrURL); ok {
		useSiteMail = true
		siteMailTo = to
	}

	if r.Method == http.MethodPost {
		if err := r.ParseForm(); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		row.Name = formString(r, "name")
		row.Status = formStatus(r)
		if owner != nil {
			row.UserID = owner.UserID
		} else if userID, ok := parseID(r.FormValue("user_id")); ok {
			row.UserID = userID
		} else {
			row.UserID = 0
		}
		selectedEvents = r.Form["events"]
		templates = notificationTemplatesFromForm(r)
		row.TemplatesJSON = notifications.TemplatesToJSON(templates)
		useSiteMail = r.FormValue("shoutrr_source") == "site_mail"
		siteMailTo = formString(r, "site_mail_to")
		if useSiteMail {
			built, err := mail.ShoutrrSMTPURL(mailCfg, siteMailTo)
			if err != nil {
				h.renderNotificationForm(w, r, owner, base, row, selectedEvents, templates, useSiteMail, siteMailTo, isNew, err.Error())
				return
			}
			row.ShoutrrURL = built
		} else {
			row.ShoutrrURL = formString(r, "shoutrr_url")
		}
		if row.UserID == 0 {
			h.renderNotificationForm(w, r, owner, base, row, selectedEvents, templates, useSiteMail, siteMailTo, isNew, "Choose the account this notification belongs to.")
			return
		}
		var account models.User
		if err := db.First(&account, row.UserID).Error; err != nil {
			h.renderNotificationForm(w, r, owner, base, row, selectedEvents, templates, useSiteMail, siteMailTo, isNew, "Account not found.")
			return
		}
		if row.Name == "" || row.ShoutrrURL == "" {
			h.renderNotificationForm(w, r, owner, base, row, selectedEvents, templates, useSiteMail, siteMailTo, isNew, "Name and shoutrrr URL are required.")
			return
		}
		var saveErr error
		if isNew {
			saveErr = db.Create(&row).Error
		} else {
			saveErr = db.Save(&row).Error
		}
		if saveErr != nil {
			h.renderNotificationForm(w, r, owner, base, row, selectedEvents, templates, useSiteMail, siteMailTo, isNew, saveErr.Error())
			return
		}
		if err := replaceNotificationEvents(db, row.NotificationID, r.Form["events"]); err != nil {
			h.renderNotificationForm(w, r, owner, base, row, selectedEvents, templates, useSiteMail, siteMailTo, isNew, err.Error())
			return
		}
		editorBase := userNotificationsBase(row.UserID)
		redirectList(w, r, editorBase+"/"+strconv.FormatUint(uint64(row.NotificationID), 10)+"?saved=1")
		return
	}
	if isNew && owner != nil && strings.TrimSpace(siteMailTo) == "" {
		siteMailTo = strings.TrimSpace(owner.Email)
	}
	h.renderNotificationForm(w, r, owner, base, row, selectedEvents, templates, useSiteMail, siteMailTo, isNew, notificationFormErrorMessage(r, ""))
}

func notificationTemplatesFromForm(r *http.Request) map[string]notifications.EventTemplate {
	out := map[string]notifications.EventTemplate{}
	for _, event := range notifications.NotificationEventsFor(r.Context()) {
		subject := strings.TrimSpace(r.FormValue("template_subject_" + event))
		body := strings.TrimSpace(r.FormValue("template_body_" + event))
		if subject == "" && body == "" {
			continue
		}
		out[event] = notifications.EventTemplate{Subject: subject, Body: body}
	}
	return out
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

func (h *Handler) renderNotificationForm(w http.ResponseWriter, r *http.Request, owner *models.User, base string, row models.Notification, selected []string, templates map[string]notifications.EventTemplate, useSiteMail bool, siteMailTo string, isNew bool, errMsg string) {
	title := "Add Notification"
	if !isNew {
		title = "Edit Notification"
	}
	if templates == nil {
		templates = map[string]notifications.EventTemplate{}
	}
	mailCfg, _ := mail.LoadSettings(r.Context())
	db, _ := sites.DB(r.Context())
	activeNav := "notifications"
	if owner != nil || row.UserID > 0 {
		activeNav = "accounts"
	}
	if base == "" {
		base = notificationsBase
	}
	data := formData(map[string]any{
		"ActiveNav":          activeNav,
		"Row":                row,
		"IsNew":              isNew,
		"BasePath":           base,
		"LockOwner":          owner != nil,
		"Owner":              owner,
		"Accounts":           notificationAccountChoices(db),
		"SelectedEvents":     selected,
		"HookEvents":         notifications.NotificationEventsFor(r.Context()),
		"SubscribableGroups": notifications.EventGroupsFor(r.Context()),
		"Templates":          templates,
		"TemplateEvents":     notifications.NotificationEventsFor(r.Context()),
		"EventPlaceholders":  notificationPlaceholdersMap(r.Context()),
		"LastSentLabel":      formatNotificationTime(row.LastSentAt),
		"LastErrorLabel":     formatNotificationTime(row.LastErrorAt),
		"MailConfigured":     mailCfg.Configured(),
		"MailHostLabel":      strings.TrimSpace(mailCfg.Host),
		"MailFromLabel":      strings.TrimSpace(mailCfg.FromEmail),
		"UseSiteMail":        useSiteMail,
		"SiteMailTo":         siteMailTo,
		"Success":            notificationFormSuccessMessage(r),
	})
	if !isNew && row.NotificationID > 0 {
		data["PageSecondaryActionPostURL"] = base + "/" + strconv.FormatUint(uint64(row.NotificationID), 10) + "/test"
		data["PageSecondaryActionLabel"] = "Test Now"
		data["PageSecondaryActionIcon"] = "bi-send"
	}
	if errMsg != "" {
		data["Error"] = errMsg
	}
	h.render(w, r, title, "admin/notifications_form.html", data)
}

func notificationFormSuccessMessage(r *http.Request) string {
	if r == nil {
		return ""
	}
	switch {
	case r.URL.Query().Get("tested") == "1":
		return "Test message sent."
	case r.URL.Query().Get("saved") == "1":
		return "Notification saved. Use Test Now to verify delivery."
	default:
		return ""
	}
}

func notificationFormErrorMessage(r *http.Request, errMsg string) string {
	if strings.TrimSpace(errMsg) != "" {
		return errMsg
	}
	if r != nil && r.URL.Query().Get("test_error") == "1" {
		return "Test message failed. See delivery error below."
	}
	return ""
}

func (h *Handler) notificationTest(w http.ResponseWriter, r *http.Request, owner *models.User, base, idStr string) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	id, ok := parseID(idStr)
	if !ok {
		h.notFound(w, r)
		return
	}
	if !notificationOwnedBy(r, owner, id) {
		h.notFound(w, r)
		return
	}
	_ = r.ParseForm()
	dest := base + "/" + idStr
	if strings.TrimSpace(r.FormValue("return")) == "list" {
		dest = base
	}
	if err := notifications.SendTest(r.Context(), id); err != nil {
		redirectList(w, r, dest+"?test_error=1")
		return
	}
	redirectList(w, r, dest+"?tested=1")
}

func (h *Handler) notificationDelete(w http.ResponseWriter, r *http.Request, owner *models.User, base, idStr string) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	id, ok := parseID(idStr)
	if !ok {
		h.notFound(w, r)
		return
	}
	if !notificationOwnedBy(r, owner, id) {
		h.notFound(w, r)
		return
	}
	db, _ := sites.DB(r.Context())
	db.Where("notification_id = ?", id).Delete(&models.NotificationEvent{})
	if err := db.Delete(&models.Notification{}, id).Error; err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	redirectList(w, r, base)
}

func (h *Handler) notificationToggleStatus(w http.ResponseWriter, r *http.Request, owner *models.User, base, idStr string) {
	id, ok := parseID(idStr)
	if !ok || !notificationOwnedBy(r, owner, id) {
		h.notFound(w, r)
		return
	}
	h.postToggleModel(w, r, idStr, &models.Notification{}, base, nil)
}

func notificationOwnedBy(r *http.Request, owner *models.User, id uint) bool {
	if owner == nil {
		return true
	}
	db, err := sites.DB(r.Context())
	if err != nil {
		return false
	}
	var row models.Notification
	if err := db.Select("notification_id", "user_id").First(&row, id).Error; err != nil {
		return false
	}
	return row.UserID == owner.UserID
}

func notificationOwnerLabel(u models.User) string {
	name := strings.TrimSpace(strings.TrimSpace(u.GivenName + " " + u.FamilyName))
	if name == "" {
		name = u.Username
	}
	if email := strings.TrimSpace(u.Email); email != "" && email != name {
		return name + " (" + email + ")"
	}
	return name
}

func notificationOwnerNames(db *gorm.DB, rows []models.Notification) map[uint]string {
	out := map[uint]string{}
	if db == nil || len(rows) == 0 {
		return out
	}
	ids := make([]uint, 0, len(rows))
	seen := map[uint]struct{}{}
	for _, row := range rows {
		if row.UserID == 0 {
			continue
		}
		if _, ok := seen[row.UserID]; ok {
			continue
		}
		seen[row.UserID] = struct{}{}
		ids = append(ids, row.UserID)
	}
	if len(ids) == 0 {
		return out
	}
	var users []models.User
	if err := db.Where("user_id IN ?", ids).Find(&users).Error; err != nil {
		return out
	}
	for _, u := range users {
		out[u.UserID] = notificationOwnerLabel(u)
	}
	return out
}

func notificationAccountChoices(db *gorm.DB) []models.User {
	if db == nil {
		return nil
	}
	var users []models.User
	db.Where("status = ?", models.StatusActive).Order("username asc").Find(&users)
	return users
}

func notificationPlaceholdersMap(ctx context.Context) map[string][]extension.NotificationPlaceholder {
	out := map[string][]extension.NotificationPlaceholder{}
	infos, err := notifications.ExtensionEventInfos(ctx)
	if err != nil {
		return out
	}
	for _, info := range infos {
		if len(info.Placeholders) > 0 {
			out[info.Event] = info.Placeholders
		}
	}
	return out
}
