package models

import "time"

// APICredential identifies a headless client application allowed to call the Content API.
type APICredential struct {
	CredentialID uint       `gorm:"primaryKey"`
	Name         string     `gorm:"size:128;not null"`
	TokenPrefix  string     `gorm:"size:16;not null;index"`
	TokenHash    string     `gorm:"size:128;not null"`
	Status       Status     `gorm:"size:16;not null;default:active"`
	ExpiresAt    *time.Time
	LastUsedAt   *time.Time
	CreatedBy    *uint
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// APIPendingToken stores short-lived opaque state for multi-step API flows.
type APIPendingToken struct {
	PendingID uint      `gorm:"primaryKey"`
	TokenHash string    `gorm:"size:128;uniqueIndex;not null"`
	Kind      string    `gorm:"size:32;not null;index"`
	UserID    uint      `gorm:"index"`
	Payload   string    `gorm:"type:text"`
	ExpiresAt time.Time `gorm:"index"`
	CreatedAt time.Time
}
