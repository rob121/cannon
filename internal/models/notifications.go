package models

import "time"

// Notification sends messages via a shoutrrr service URL when hooks fire.
type Notification struct {
	NotificationID uint   `gorm:"primaryKey"`
	Name           string `gorm:"size:128;not null"`
	ShoutrrURL     string `gorm:"size:1024;not null"`
	Status         Status `gorm:"size:16;not null;default:active"`
	CreatedAt      time.Time
	UpdatedAt      time.Time
	Events         []NotificationEvent `gorm:"foreignKey:NotificationID;constraint:OnDelete:CASCADE"`
}

// NotificationEvent binds a hook name to a notification.
type NotificationEvent struct {
	NotificationID uint   `gorm:"primaryKey"`
	Event          string `gorm:"primaryKey;size:64"`
}

const NotificationChannelEmail = "email"

// NotificationSubscription is a Layer 2 per-user or per-role hook subscription.
type NotificationSubscription struct {
	SubscriptionID uint   `gorm:"primaryKey"`
	UserID         *uint  `gorm:"index;uniqueIndex:idx_notif_sub_user,priority:1"`
	RoleID         *uint  `gorm:"index;uniqueIndex:idx_notif_sub_role,priority:1"`
	Event          string `gorm:"size:64;not null;index;uniqueIndex:idx_notif_sub_user,priority:2;uniqueIndex:idx_notif_sub_role,priority:2"`
	Channel        string `gorm:"size:16;not null;default:email;uniqueIndex:idx_notif_sub_user,priority:3;uniqueIndex:idx_notif_sub_role,priority:3"`
	Status         Status `gorm:"size:16;not null;default:active"`
	FiltersJSON    string `gorm:"type:text"`
	CreatedAt      time.Time
	UpdatedAt      time.Time
}
