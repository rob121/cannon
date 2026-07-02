package auth

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"time"

	"github.com/rob121/cannon/internal/models"
	"github.com/rob121/cannon/internal/routepath"
	"github.com/rob121/cannon/internal/sites"
	"gorm.io/gorm"
)

const (
	TokenVerify = "verify"
	TokenReset  = "reset"
)

const (
	verifyTokenTTL = 7 * 24 * time.Hour
	resetTokenTTL  = 2 * time.Hour
)

var ErrTokenInvalid = errors.New("invalid or expired token")

func newTokenValue() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}

// EnsureVerifyToken returns a valid verification token, creating one when needed.
func EnsureVerifyToken(ctx context.Context, userID uint) (string, error) {
	db, err := sites.DB(ctx)
	if err != nil {
		return "", err
	}
	var row models.UserToken
	err = db.Where("user_id = ? AND type = ? AND used_at IS NULL AND expires_at > ?", userID, TokenVerify, time.Now()).
		Order("user_token_id desc").First(&row).Error
	if err == nil {
		return row.Token, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return "", err
	}
	return IssueVerifyToken(ctx, userID)
}

// IssueVerifyToken creates a verification token for a user.
func IssueVerifyToken(ctx context.Context, userID uint) (string, error) {
	return issueToken(ctx, userID, TokenVerify, verifyTokenTTL)
}

// IssueResetToken creates a password reset token for a user.
func IssueResetToken(ctx context.Context, userID uint) (string, error) {
	return issueToken(ctx, userID, TokenReset, resetTokenTTL)
}

func issueToken(ctx context.Context, userID uint, tokenType string, ttl time.Duration) (string, error) {
	db, err := sites.DB(ctx)
	if err != nil {
		return "", err
	}
	value, err := newTokenValue()
	if err != nil {
		return "", err
	}
	row := models.UserToken{
		UserID:    userID,
		Type:      tokenType,
		Token:     value,
		ExpiresAt: time.Now().Add(ttl),
	}
	if err := db.Create(&row).Error; err != nil {
		return "", err
	}
	return value, nil
}

// ConsumeToken validates and marks a token used.
func ConsumeToken(ctx context.Context, tokenType, value string) (*models.UserToken, error) {
	db, err := sites.DB(ctx)
	if err != nil {
		return nil, err
	}
	var row models.UserToken
	if err := db.Where("token = ? AND type = ?", value, tokenType).First(&row).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrTokenInvalid
		}
		return nil, err
	}
	if row.UsedAt != nil || time.Now().After(row.ExpiresAt) {
		return nil, ErrTokenInvalid
	}
	now := time.Now()
	row.UsedAt = &now
	if err := db.Save(&row).Error; err != nil {
		return nil, err
	}
	return &row, nil
}

// LookupToken returns a valid unused token without consuming it.
func LookupToken(ctx context.Context, tokenType, value string) (*models.UserToken, error) {
	db, err := sites.DB(ctx)
	if err != nil {
		return nil, err
	}
	var row models.UserToken
	if err := db.Where("token = ? AND type = ?", value, tokenType).First(&row).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrTokenInvalid
		}
		return nil, err
	}
	if row.UsedAt != nil || time.Now().After(row.ExpiresAt) {
		return nil, ErrTokenInvalid
	}
	return &row, nil
}

// VerifyURL builds a frontend verification URL for a token value.
func VerifyURL(ctx context.Context, token string) string {
	return routepath.ControllerWithSuffix(ctx, "auth", "verify", token)
}

// ResetURL builds a frontend password reset URL for a token value.
func ResetURL(ctx context.Context, token string) string {
	return routepath.ControllerWithSuffix(ctx, "auth", "reset-submit", token)
}
