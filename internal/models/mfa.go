package models

import "time"

// UserTOTP stores a time-based one-time password secret for a user.
type UserTOTP struct {
	UserID    uint      `gorm:"primaryKey"`
	Secret    string    `gorm:"size:128;not null"`
	Enabled   bool      `gorm:"not null;default:false"`
	CreatedAt time.Time `gorm:"autoCreateTime"`
}

// UserPasskey stores a WebAuthn credential for passwordless or second-factor sign-in.
type UserPasskey struct {
	PasskeyID      uint       `gorm:"primaryKey"`
	UserID         uint       `gorm:"index;not null"`
	Name           string     `gorm:"size:128;not null"`
	CredentialJSON string     `gorm:"type:text;not null"`
	CreatedAt      time.Time  `gorm:"autoCreateTime"`
	LastUsedAt     *time.Time `gorm:"index"`
}
