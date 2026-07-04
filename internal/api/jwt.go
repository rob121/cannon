package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/rob121/cannon/internal/models"
	"github.com/rob121/cannon/internal/settings"
	"github.com/rob121/cannon/internal/sites"
)

const (
	settingJWTSecret       = "api_jwt_secret"
	settingJWTSecretPrev   = "api_jwt_secret_prev"
	sectionAPI             = "api"
	claimTypeAccess        = "access"
	claimTypeRefresh       = "refresh"
	defaultAccessTTL       = time.Hour
	defaultRefreshTTL      = 7 * 24 * time.Hour
)

type accessClaims struct {
	jwt.RegisteredClaims
	Typ string `json:"typ"`
	SID string `json:"sid"`
}

// IssueAccessToken creates a signed JWT for a user.
func IssueAccessToken(ctx context.Context, userID uint) (string, int, error) {
	secret, err := EnsureJWTSecret(ctx)
	if err != nil {
		return "", 0, err
	}
	ttl, err := JWTTTL(ctx)
	if err != nil {
		return "", 0, err
	}
	site, err := sites.FromContext(ctx)
	if err != nil {
		return "", 0, err
	}
	now := time.Now()
	claims := accessClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   fmt.Sprintf("%d", userID),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
			ID:        mustRandomID(),
		},
		Typ: claimTypeAccess,
		SID: site.ID,
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString([]byte(secret))
	if err != nil {
		return "", 0, err
	}
	return signed, int(ttl.Seconds()), nil
}

// IssueRefreshToken creates a long-lived refresh token stored as pending state.
func IssueRefreshToken(ctx context.Context, userID uint) (string, error) {
	ttl, err := RefreshTTL(ctx)
	if err != nil {
		return "", err
	}
	return IssuePendingToken(ctx, PendingRefresh, userID, nil, ttl)
}

// ParseAccessToken validates a JWT and returns the user id.
func ParseAccessToken(ctx context.Context, tokenStr string) (uint, error) {
	tokenStr = strings.TrimSpace(tokenStr)
	if tokenStr == "" {
		return 0, ErrInvalidJWT
	}
	secrets, err := jwtSecrets(ctx)
	if err != nil {
		return 0, err
	}
	site, err := sites.FromContext(ctx)
	if err != nil {
		return 0, err
	}
	var lastErr error
	for _, secret := range secrets {
		claims := &accessClaims{}
		token, err := jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (any, error) {
			if t.Method != jwt.SigningMethodHS256 {
				return nil, fmt.Errorf("unexpected signing method")
			}
			return []byte(secret), nil
		})
		if err != nil || !token.Valid {
			lastErr = err
			continue
		}
		if claims.Typ != claimTypeAccess {
			return 0, ErrInvalidJWT
		}
		if claims.SID != "" && claims.SID != site.ID {
			return 0, ErrInvalidJWT
		}
		var userID uint
		if _, err := fmt.Sscanf(claims.Subject, "%d", &userID); err != nil || userID == 0 {
			return 0, ErrInvalidJWT
		}
		return userID, nil
	}
	if lastErr != nil {
		return 0, ErrInvalidJWT
	}
	return 0, ErrInvalidJWT
}

// EnsureJWTSecret returns the site JWT secret, generating one when missing.
func EnsureJWTSecret(ctx context.Context) (string, error) {
	secrets, err := jwtSecrets(ctx)
	if err != nil {
		return "", err
	}
	if len(secrets) > 0 && secrets[0] != "" {
		return secrets[0], nil
	}
	buf, err := ensureJWTSecretBytes()
	if err != nil {
		return "", err
	}
	hexSecret := secretHex(buf)
	if err := saveJWTSecret(ctx, hexSecret, ""); err != nil {
		return "", err
	}
	return hexSecret, nil
}

// RotateJWTSecret generates a new secret and keeps the previous for validation during rotation.
func RotateJWTSecret(ctx context.Context) error {
	current, _ := settings.GlobalString(ctx, sectionAPI, settingJWTSecret)
	buf, err := ensureJWTSecretBytes()
	if err != nil {
		return err
	}
	return saveJWTSecret(ctx, secretHex(buf), current)
}

func saveJWTSecret(ctx context.Context, current, previous string) error {
	store := settings.NewStore()
	data, err := store.Load(ctx, settings.ScopeGlobal, sectionAPI)
	if err != nil {
		return err
	}
	if data == nil {
		data = map[string]any{}
	}
	data[settingJWTSecret] = current
	if strings.TrimSpace(previous) != "" {
		data[settingJWTSecretPrev] = previous
	} else {
		delete(data, settingJWTSecretPrev)
	}
	return store.Save(ctx, settings.ScopeGlobal, sectionAPI, data)
}

func jwtSecrets(ctx context.Context) ([]string, error) {
	current, err := settings.GlobalString(ctx, sectionAPI, settingJWTSecret)
	if err != nil {
		return nil, err
	}
	prev, _ := settings.GlobalString(ctx, sectionAPI, settingJWTSecretPrev)
	var out []string
	if strings.TrimSpace(current) != "" {
		out = append(out, current)
	}
	if strings.TrimSpace(prev) != "" && prev != current {
		out = append(out, prev)
	}
	return out, nil
}

// JWTTTL returns configured access token lifetime.
func JWTTTL(ctx context.Context) (time.Duration, error) {
	sec, err := settings.GlobalIntDefault(ctx, sectionAPI, "jwt_ttl_seconds", int(defaultAccessTTL.Seconds()))
	if err != nil {
		return defaultAccessTTL, err
	}
	if sec < 300 {
		sec = 300
	}
	return time.Duration(sec) * time.Second, nil
}

// RefreshTTL returns configured refresh token lifetime.
func RefreshTTL(ctx context.Context) (time.Duration, error) {
	sec, err := settings.GlobalIntDefault(ctx, sectionAPI, "refresh_ttl_seconds", int(defaultRefreshTTL.Seconds()))
	if err != nil {
		return defaultRefreshTTL, err
	}
	if sec < 3600 {
		sec = 3600
	}
	return time.Duration(sec) * time.Second, nil
}

func extractBearerJWT(r *http.Request) string {
	auth := strings.TrimSpace(r.Header.Get("Authorization"))
	if !strings.HasPrefix(strings.ToLower(auth), "bearer ") {
		return ""
	}
	tok := strings.TrimSpace(auth[7:])
	if strings.HasPrefix(tok, liveTokenPrefix) {
		return ""
	}
	return tok
}

func mustRandomID() string {
	raw, err := randomToken(16)
	if err != nil {
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	return raw
}

var ErrInvalidJWT = errors.New("invalid jwt")

// RefreshUserID validates a refresh token and returns the user id.
func RefreshUserID(ctx context.Context, raw string) (uint, error) {
	row, err := ConsumePendingToken(ctx, PendingRefresh, raw)
	if err != nil {
		return 0, err
	}
	return row.UserID, nil
}

// SettingsData loads api settings as map.
func SettingsData(ctx context.Context) (map[string]any, error) {
	return settings.NewStore().Load(ctx, settings.ScopeGlobal, sectionAPI)
}

// SaveSettingsData persists api settings.
func SaveSettingsData(ctx context.Context, data map[string]any) error {
	existing, err := settings.NewStore().Load(ctx, settings.ScopeGlobal, sectionAPI)
	if err != nil {
		return err
	}
	if existing == nil {
		existing = map[string]any{}
	}
	for k, v := range data {
		existing[k] = v
	}
	// never overwrite jwt secret from form
	if _, ok := existing[settingJWTSecret]; !ok {
		if _, err := EnsureJWTSecret(ctx); err != nil {
			return err
		}
		reloaded, _ := settings.NewStore().Load(ctx, settings.ScopeGlobal, sectionAPI)
		if reloaded != nil {
			existing[settingJWTSecret] = reloaded[settingJWTSecret]
		}
	}
	return settings.NewStore().Save(ctx, settings.ScopeGlobal, sectionAPI, existing)
}

// CORSOrigins returns allowed browser origins.
func CORSOrigins(ctx context.Context) ([]string, error) {
	raw, err := settings.GlobalString(ctx, sectionAPI, "cors_origins")
	if err != nil {
		return nil, err
	}
	var out []string
	for _, part := range strings.Split(raw, ",") {
		part = strings.TrimSpace(part)
		if part != "" {
			out = append(out, part)
		}
	}
	return out, nil
}

// PendingPayload decodes pending token JSON payload.
func PendingPayload(row *models.APIPendingToken, dst any) error {
	if row == nil || strings.TrimSpace(row.Payload) == "" {
		return nil
	}
	return json.Unmarshal([]byte(row.Payload), dst)
}
