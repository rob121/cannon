package api

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"time"

	"github.com/rob121/cannon/internal/models"
	"github.com/rob121/cannon/internal/sites"
	"gorm.io/gorm"
)

const (
	PendingMFALogin        = "mfa_login"
	PendingTOTPSetup       = "totp_setup"
	PendingPasskeyRegister = "passkey_register"
	PendingPasskeyLogin    = "passkey_login"
	PendingRefresh         = "refresh"
)

// IssuePendingToken stores opaque multi-step state and returns the raw token.
func IssuePendingToken(ctx context.Context, kind string, userID uint, payload any, ttl time.Duration) (string, error) {
	raw, err := randomToken(32)
	if err != nil {
		return "", err
	}
	var payloadJSON string
	if payload != nil {
		b, err := json.Marshal(payload)
		if err != nil {
			return "", err
		}
		payloadJSON = string(b)
	}
	db, err := sites.DB(ctx)
	if err != nil {
		return "", err
	}
	row := models.APIPendingToken{
		TokenHash: hashToken(raw),
		Kind:      kind,
		UserID:    userID,
		Payload:   payloadJSON,
		ExpiresAt: time.Now().Add(ttl),
	}
	if err := db.Create(&row).Error; err != nil {
		return "", err
	}
	return raw, nil
}

// ConsumePendingToken validates and deletes a pending token.
func ConsumePendingToken(ctx context.Context, kind, raw string) (*models.APIPendingToken, error) {
	db, err := sites.DB(ctx)
	if err != nil {
		return nil, err
	}
	var row models.APIPendingToken
	err = db.Where("token_hash = ? AND kind = ?", hashToken(raw), kind).First(&row).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrInvalidPendingToken
		}
		return nil, err
	}
	if time.Now().After(row.ExpiresAt) {
		_ = db.Delete(&row).Error
		return nil, ErrInvalidPendingToken
	}
	if err := db.Delete(&row).Error; err != nil {
		return nil, err
	}
	return &row, nil
}

// LookupPendingToken returns a valid pending token without consuming it.
func LookupPendingToken(ctx context.Context, kind, raw string) (*models.APIPendingToken, error) {
	db, err := sites.DB(ctx)
	if err != nil {
		return nil, err
	}
	var row models.APIPendingToken
	err = db.Where("token_hash = ? AND kind = ?", hashToken(raw), kind).First(&row).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrInvalidPendingToken
		}
		return nil, err
	}
	if time.Now().After(row.ExpiresAt) {
		return nil, ErrInvalidPendingToken
	}
	return &row, nil
}

// UpdatePendingPayload updates payload on an existing pending token.
func UpdatePendingPayload(ctx context.Context, pendingID uint, payload any) error {
	b, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	db, err := sites.DB(ctx)
	if err != nil {
		return err
	}
	return db.Model(&models.APIPendingToken{}).Where("pending_id = ?", pendingID).
		Update("payload", string(b)).Error
}

// PurgeExpiredPending removes expired pending tokens.
func PurgeExpiredPending(ctx context.Context) error {
	db, err := sites.DB(ctx)
	if err != nil {
		return err
	}
	return db.Where("expires_at < ?", time.Now()).Delete(&models.APIPendingToken{}).Error
}

var ErrInvalidPendingToken = errors.New("invalid or expired token")

func ensureJWTSecretBytes() ([]byte, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return nil, err
	}
	return buf, nil
}

func secretHex(b []byte) string {
	return hex.EncodeToString(b)
}
