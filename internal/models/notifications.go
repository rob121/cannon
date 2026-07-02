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
