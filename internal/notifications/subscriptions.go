package notifications

import (
	"context"
	"strings"

	"github.com/rob121/cannon/internal/models"
	"github.com/rob121/cannon/internal/sites"
	"gorm.io/gorm"
)

// UserProfileState drives the account notifications UI.
type UserProfileState struct {
	Groups       []EventGroup
	Checked      map[string]bool
	RoleDefaults map[string]bool
}

// UserProfileState loads effective subscription checkboxes for a user.
func LoadUserProfileState(ctx context.Context, userID uint) (UserProfileState, error) {
	db, err := sites.DB(ctx)
	if err != nil {
		return UserProfileState{}, err
	}
	roleDefaults, err := roleDefaultEventsForUser(db, userID)
	if err != nil {
		return UserProfileState{}, err
	}
	userSubs, err := userSubscriptionMap(db, userID)
	if err != nil {
		return UserProfileState{}, err
	}
	checked := map[string]bool{}
	for _, event := range SubscribableEvents {
		if row, ok := userSubs[event]; ok {
			checked[event] = row.Status == models.StatusActive
			continue
		}
		checked[event] = roleDefaults[event]
	}
	return UserProfileState{
		Groups:       EventGroups(),
		Checked:      checked,
		RoleDefaults: roleDefaults,
	}, nil
}

// SaveUserSubscriptions persists per-user event preferences.
func SaveUserSubscriptions(ctx context.Context, userID uint, selected []string) error {
	db, err := sites.DB(ctx)
	if err != nil {
		return err
	}
	roleDefaults, err := roleDefaultEventsForUser(db, userID)
	if err != nil {
		return err
	}
	selectedSet := map[string]struct{}{}
	for _, event := range selected {
		event = strings.TrimSpace(event)
		if !isSubscribableEvent(event) {
			continue
		}
		selectedSet[event] = struct{}{}
	}
	for _, event := range SubscribableEvents {
		_, wants := selectedSet[event]
		hasRoleDefault := roleDefaults[event]
		switch {
		case wants:
			if err := upsertUserSubscription(db, userID, event, models.StatusActive); err != nil {
				return err
			}
		case hasRoleDefault:
			if err := upsertUserSubscription(db, userID, event, models.StatusInactive); err != nil {
				return err
			}
		default:
			if err := db.Where("user_id = ? AND event = ? AND channel = ?", userID, event, models.NotificationChannelEmail).
				Delete(&models.NotificationSubscription{}).Error; err != nil {
				return err
			}
		}
	}
	return nil
}

// RoleSubscriptionEvents returns active role-default events.
func RoleSubscriptionEvents(db *gorm.DB, roleID uint) ([]string, error) {
	var rows []models.NotificationSubscription
	if err := db.Where("role_id = ? AND channel = ? AND status = ?", roleID, models.NotificationChannelEmail, models.StatusActive).
		Order("event asc").Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]string, 0, len(rows))
	for _, row := range rows {
		out = append(out, row.Event)
	}
	return out, nil
}

// ReplaceRoleSubscriptions saves role-default notification events.
func ReplaceRoleSubscriptions(db *gorm.DB, roleID uint, events []string) error {
	if err := db.Where("role_id = ?", roleID).Delete(&models.NotificationSubscription{}).Error; err != nil {
		return err
	}
	for _, event := range events {
		event = strings.TrimSpace(event)
		if !isSubscribableEvent(event) {
			continue
		}
		roleIDCopy := roleID
		row := models.NotificationSubscription{
			RoleID:  &roleIDCopy,
			Event:   event,
			Channel: models.NotificationChannelEmail,
			Status:  models.StatusActive,
		}
		if err := db.Create(&row).Error; err != nil {
			return err
		}
	}
	return nil
}

// DeleteRoleSubscriptions removes subscriptions for a deleted role.
func DeleteRoleSubscriptions(db *gorm.DB, roleID uint) error {
	return db.Where("role_id = ?", roleID).Delete(&models.NotificationSubscription{}).Error
}

type recipient struct {
	UserID uint
	Email  string
}

func resolveEmailRecipients(ctx context.Context, event string) ([]recipient, error) {
	db, err := sites.DB(ctx)
	if err != nil {
		return nil, err
	}
	roleUserIDs, err := usersForRoleSubscriptions(db, event)
	if err != nil {
		return nil, err
	}
	userSubs, err := allUserSubscriptionsForEvent(db, event)
	if err != nil {
		return nil, err
	}
	optOut := map[uint]struct{}{}
	optIn := map[uint]struct{}{}
	for userID, status := range userSubs {
		switch status {
		case models.StatusActive:
			optIn[userID] = struct{}{}
		case models.StatusInactive:
			optOut[userID] = struct{}{}
		}
	}
	ids := map[uint]struct{}{}
	for id := range roleUserIDs {
		if _, skip := optOut[id]; skip {
			continue
		}
		ids[id] = struct{}{}
	}
	for id := range optIn {
		ids[id] = struct{}{}
	}
	if len(ids) == 0 {
		return nil, nil
	}
	userIDs := make([]uint, 0, len(ids))
	for id := range ids {
		userIDs = append(userIDs, id)
	}
	var users []models.User
	if err := db.Where("user_id IN ? AND status = ? AND locked = ? AND email <> ''", userIDs, models.StatusActive, false).
		Find(&users).Error; err != nil {
		return nil, err
	}
	seen := map[string]struct{}{}
	out := make([]recipient, 0, len(users))
	for _, u := range users {
		email := strings.TrimSpace(strings.ToLower(u.Email))
		if email == "" {
			continue
		}
		if _, ok := seen[email]; ok {
			continue
		}
		seen[email] = struct{}{}
		out = append(out, recipient{UserID: u.UserID, Email: strings.TrimSpace(u.Email)})
	}
	return out, nil
}

func usersForRoleSubscriptions(db *gorm.DB, event string) (map[uint]struct{}, error) {
	var roleIDs []uint
	if err := db.Model(&models.NotificationSubscription{}).
		Where("role_id IS NOT NULL AND event = ? AND channel = ? AND status = ?", event, models.NotificationChannelEmail, models.StatusActive).
		Pluck("role_id", &roleIDs).Error; err != nil {
		return nil, err
	}
	if len(roleIDs) == 0 {
		return map[uint]struct{}{}, nil
	}
	out := map[uint]struct{}{}
	for _, roleID := range roleIDs {
		ids, err := userIDsWithRole(db, roleID)
		if err != nil {
			return nil, err
		}
		for id := range ids {
			out[id] = struct{}{}
		}
	}
	return out, nil
}

func userIDsWithRole(db *gorm.DB, roleID uint) (map[uint]struct{}, error) {
	out := map[uint]struct{}{}
	var direct []uint
	if err := db.Table("user_roles").Where("role_role_id = ?", roleID).Pluck("user_user_id", &direct).Error; err != nil {
		return nil, err
	}
	for _, id := range direct {
		out[id] = struct{}{}
	}
	var viaGroup []uint
	err := db.Table("user_groups").
		Select("DISTINCT user_groups.user_user_id").
		Joins("JOIN group_roles ON group_roles.group_group_id = user_groups.group_group_id").
		Where("group_roles.role_role_id = ?", roleID).
		Pluck("user_user_id", &viaGroup).Error
	if err != nil {
		return nil, err
	}
	for _, id := range viaGroup {
		out[id] = struct{}{}
	}
	return out, nil
}

func roleDefaultEventsForUser(db *gorm.DB, userID uint) (map[string]bool, error) {
	roleIDs, err := roleIDsForUser(db, userID)
	if err != nil {
		return nil, err
	}
	out := map[string]bool{}
	if len(roleIDs) == 0 {
		return out, nil
	}
	var rows []models.NotificationSubscription
	if err := db.Where("role_id IN ? AND channel = ? AND status = ?", roleIDs, models.NotificationChannelEmail, models.StatusActive).
		Find(&rows).Error; err != nil {
		return nil, err
	}
	for _, row := range rows {
		out[row.Event] = true
	}
	return out, nil
}

func roleIDsForUser(db *gorm.DB, userID uint) ([]uint, error) {
	seen := map[uint]struct{}{}
	var direct []uint
	if err := db.Table("user_roles").Where("user_user_id = ?", userID).Pluck("role_role_id", &direct).Error; err != nil {
		return nil, err
	}
	for _, id := range direct {
		seen[id] = struct{}{}
	}
	var viaGroup []uint
	if err := db.Table("user_groups").
		Select("DISTINCT group_roles.role_role_id").
		Joins("JOIN group_roles ON group_roles.group_group_id = user_groups.group_group_id").
		Where("user_groups.user_user_id = ?", userID).
		Pluck("role_role_id", &viaGroup).Error; err != nil {
		return nil, err
	}
	for _, id := range viaGroup {
		seen[id] = struct{}{}
	}
	out := make([]uint, 0, len(seen))
	for id := range seen {
		out = append(out, id)
	}
	return out, nil
}

func userSubscriptionMap(db *gorm.DB, userID uint) (map[string]models.NotificationSubscription, error) {
	var rows []models.NotificationSubscription
	if err := db.Where("user_id = ? AND channel = ?", userID, models.NotificationChannelEmail).Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make(map[string]models.NotificationSubscription, len(rows))
	for _, row := range rows {
		out[row.Event] = row
	}
	return out, nil
}

func allUserSubscriptionsForEvent(db *gorm.DB, event string) (map[uint]models.Status, error) {
	var rows []models.NotificationSubscription
	if err := db.Where("user_id IS NOT NULL AND event = ? AND channel = ?", event, models.NotificationChannelEmail).
		Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make(map[uint]models.Status, len(rows))
	for _, row := range rows {
		if row.UserID == nil {
			continue
		}
		out[*row.UserID] = row.Status
	}
	return out, nil
}

func upsertUserSubscription(db *gorm.DB, userID uint, event string, status models.Status) error {
	var existing models.NotificationSubscription
	err := db.Where("user_id = ? AND event = ? AND channel = ?", userID, event, models.NotificationChannelEmail).
		First(&existing).Error
	if err == gorm.ErrRecordNotFound {
		userIDCopy := userID
		return db.Create(&models.NotificationSubscription{
			UserID:  &userIDCopy,
			Event:   event,
			Channel: models.NotificationChannelEmail,
			Status:  status,
		}).Error
	}
	if err != nil {
		return err
	}
	existing.Status = status
	return db.Save(&existing).Error
}
