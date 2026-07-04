package api

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/rob121/cannon/internal/models"
	"github.com/rob121/cannon/internal/sites"
	"gorm.io/gorm"
)

const liveTokenPrefix = "cn_live_"

// IssueCredential creates a new API credential and returns the raw token once.
func IssueCredential(ctx context.Context, name string, expiresAt *time.Time, createdBy *uint) (models.APICredential, string, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return models.APICredential{}, "", errors.New("name is required")
	}
	raw, err := randomToken(24)
	if err != nil {
		return models.APICredential{}, "", err
	}
	token := liveTokenPrefix + raw
	prefix := token[:16]
	row := models.APICredential{
		Name:        name,
		TokenPrefix: prefix,
		TokenHash:   hashToken(token),
		Status:      models.StatusActive,
		ExpiresAt:   expiresAt,
		CreatedBy:   createdBy,
	}
	db, err := sites.DB(ctx)
	if err != nil {
		return models.APICredential{}, "", err
	}
	if err := db.Create(&row).Error; err != nil {
		return models.APICredential{}, "", err
	}
	return row, token, nil
}

// RotateCredential invalidates the old hash and issues a new token.
func RotateCredential(ctx context.Context, credentialID uint) (string, error) {
	db, err := sites.DB(ctx)
	if err != nil {
		return "", err
	}
	var row models.APICredential
	if err := db.First(&row, credentialID).Error; err != nil {
		return "", err
	}
	raw, err := randomToken(24)
	if err != nil {
		return "", err
	}
	token := liveTokenPrefix + raw
	row.TokenPrefix = token[:16]
	row.TokenHash = hashToken(token)
	if err := db.Save(&row).Error; err != nil {
		return "", err
	}
	return token, nil
}

// ValidateAPIKey checks a presented API key and returns the credential row.
func ValidateAPIKey(ctx context.Context, token string) (*models.APICredential, error) {
	token = strings.TrimSpace(token)
	if token == "" || !strings.HasPrefix(token, liveTokenPrefix) {
		return nil, ErrInvalidAPIKey
	}
	if len(token) < 20 {
		return nil, ErrInvalidAPIKey
	}
	prefix := token[:16]
	hash := hashToken(token)
	db, err := sites.DB(ctx)
	if err != nil {
		return nil, err
	}
	var row models.APICredential
	err = db.Where("token_prefix = ? AND token_hash = ?", prefix, hash).First(&row).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrInvalidAPIKey
		}
		return nil, err
	}
	if row.Status != models.StatusActive {
		return nil, ErrInvalidAPIKey
	}
	if row.ExpiresAt != nil && time.Now().After(*row.ExpiresAt) {
		return nil, ErrInvalidAPIKey
	}
	touchCredentialLastUsed(ctx, &row)
	return &row, nil
}

func touchCredentialLastUsed(ctx context.Context, row *models.APICredential) {
	if row == nil || row.CredentialID == 0 {
		return
	}
	if row.LastUsedAt != nil && time.Since(*row.LastUsedAt) < time.Minute {
		return
	}
	db, err := sites.DB(ctx)
	if err != nil {
		return
	}
	now := time.Now()
	_ = db.Model(row).Update("last_used_at", now).Error
}

func hashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

func extractAPIKey(r *http.Request) string {
	if v := strings.TrimSpace(r.Header.Get("X-Cannon-API-Key")); v != "" {
		return v
	}
	auth := strings.TrimSpace(r.Header.Get("Authorization"))
	if strings.HasPrefix(strings.ToLower(auth), "bearer ") {
		tok := strings.TrimSpace(auth[7:])
		if strings.HasPrefix(tok, liveTokenPrefix) {
			return tok
		}
	}
	return ""
}

var ErrInvalidAPIKey = errors.New("invalid api key")
