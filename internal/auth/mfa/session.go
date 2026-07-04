package mfa

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	"github.com/go-webauthn/webauthn/protocol"
	"github.com/go-webauthn/webauthn/webauthn"
	"github.com/rob121/cannon/internal/models"
	"github.com/rob121/cannon/internal/settings"
	"github.com/rob121/cannon/internal/sites"
	"github.com/rob121/cannon/internal/user"
	"gorm.io/gorm"
)

const (
	sessionPendingMFAUserID  = "mfa_pending_user_id"
	sessionPendingMFAContext = "mfa_pending_context"
	sessionPendingMFAReturn  = "mfa_pending_return"
	sessionPendingTOTPSecret = "mfa_pending_totp_secret"
	sessionWebAuthnData      = "webauthn_session_data"
)

// PendingMFA holds in-progress sign-in state before MFA completes.
type PendingMFA struct {
	UserID  uint
	Context string
	Return  string
}

// SetPendingMFA stores pending sign-in state without completing login.
func SetPendingMFA(s *user.Service, pending PendingMFA) error {
	if err := s.SetSessionValue(sessionPendingMFAUserID, pending.UserID); err != nil {
		return err
	}
	if err := s.SetSessionValue(sessionPendingMFAContext, pending.Context); err != nil {
		return err
	}
	if pending.Return != "" {
		if err := s.SetSessionValue(sessionPendingMFAReturn, pending.Return); err != nil {
			return err
		}
	}
	return nil
}

// GetPendingMFA returns pending sign-in state when present.
func GetPendingMFA(s *user.Service) (PendingMFA, bool) {
	rawID, ok := s.SessionValue(sessionPendingMFAUserID)
	if !ok {
		return PendingMFA{}, false
	}
	userID, ok := sessionUint(rawID)
	if !ok || userID == 0 {
		return PendingMFA{}, false
	}
	ctxName := "frontend"
	if rawCtx, ok := s.SessionValue(sessionPendingMFAContext); ok {
		if s, ok := rawCtx.(string); ok && s != "" {
			ctxName = s
		}
	}
	ret := ""
	if rawRet, ok := s.SessionValue(sessionPendingMFAReturn); ok {
		if s, ok := rawRet.(string); ok {
			ret = s
		}
	}
	return PendingMFA{UserID: userID, Context: ctxName, Return: ret}, true
}

// ClearPendingMFA removes pending sign-in state.
func ClearPendingMFA(s *user.Service) error {
	_ = s.ClearSessionValue(sessionPendingMFAUserID)
	_ = s.ClearSessionValue(sessionPendingMFAContext)
	_ = s.ClearSessionValue(sessionPendingMFAReturn)
	return nil
}

// SetPendingTOTPSecret stores a TOTP secret during enrollment.
func SetPendingTOTPSecret(s *user.Service, secret string) error {
	return s.SetSessionValue(sessionPendingTOTPSecret, secret)
}

// GetPendingTOTPSecret returns the enrollment secret when present.
func GetPendingTOTPSecret(s *user.Service) (string, bool) {
	raw, ok := s.SessionValue(sessionPendingTOTPSecret)
	if !ok {
		return "", false
	}
	secret, ok := raw.(string)
	return secret, ok && secret != ""
}

// ClearPendingTOTPSecret removes the enrollment secret.
func ClearPendingTOTPSecret(s *user.Service) error {
	return s.ClearSessionValue(sessionPendingTOTPSecret)
}

// SaveWebAuthnSession stores WebAuthn ceremony state.
func SaveWebAuthnSession(s *user.Service, data *webauthn.SessionData) error {
	raw, err := json.Marshal(data)
	if err != nil {
		return err
	}
	return s.SetSessionValue(sessionWebAuthnData, string(raw))
}

// LoadWebAuthnSession loads WebAuthn ceremony state.
func LoadWebAuthnSession(s *user.Service) (*webauthn.SessionData, error) {
	raw, ok := s.SessionValue(sessionWebAuthnData)
	if !ok {
		return nil, fmt.Errorf("webauthn session missing")
	}
	text, ok := raw.(string)
	if !ok || text == "" {
		return nil, fmt.Errorf("webauthn session invalid")
	}
	var data webauthn.SessionData
	if err := json.Unmarshal([]byte(text), &data); err != nil {
		return nil, err
	}
	return &data, nil
}

// ClearWebAuthnSession removes stored WebAuthn ceremony state.
func ClearWebAuthnSession(s *user.Service) error {
	return s.ClearSessionValue(sessionWebAuthnData)
}

// UserRequiresMFA reports whether the user must complete MFA at sign-in.
func UserRequiresMFA(ctx context.Context, userID uint) (bool, error) {
	totpAllowed, err := settings.AllowMFATOTP(ctx)
	if err != nil {
		return false, err
	}
	passkeysAllowed, err := settings.AllowPasskeys(ctx)
	if err != nil {
		return false, err
	}
	if !totpAllowed && !passkeysAllowed {
		return false, nil
	}
	db, err := sites.DB(ctx)
	if err != nil {
		return false, err
	}
	if totpAllowed {
		enabled, err := TOTPEnabled(db, userID)
		if err != nil {
			return false, err
		}
		if enabled {
			return true, nil
		}
	}
	if passkeysAllowed {
		count, err := PasskeyCount(db, userID)
		if err != nil {
			return false, err
		}
		if count > 0 {
			return true, nil
		}
	}
	return false, nil
}

// PasskeyCount returns how many passkeys a user has registered.
func PasskeyCount(db *gorm.DB, userID uint) (int64, error) {
	var count int64
	err := db.Model(&models.UserPasskey{}).Where("user_id = ?", userID).Count(&count).Error
	return count, err
}

func sessionUint(v any) (uint, bool) {
	switch id := v.(type) {
	case float64:
		return uint(id), true
	case int:
		return uint(id), true
	case uint:
		return id, true
	default:
		return 0, false
	}
}

// WebAuthnInstance builds a WebAuthn service for the current site.
func WebAuthnInstance(ctx context.Context) (*webauthn.WebAuthn, error) {
	site, err := sites.FromContext(ctx)
	if err != nil {
		return nil, err
	}
	displayName, _ := settings.GlobalString(ctx, settings.SectionGeneral, "site_name")
	if displayName == "" {
		displayName = site.Name
	}
	if displayName == "" {
		displayName = "Cannon"
	}
	rpID, origin, err := rpIDAndOrigin(site.Host)
	if err != nil {
		return nil, err
	}
	cfg := &webauthn.Config{
		RPID:          rpID,
		RPDisplayName: displayName,
		RPOrigins:     []string{origin},
		AuthenticatorSelection: protocol.AuthenticatorSelection{
			UserVerification: protocol.VerificationRequired,
		},
		EncodeUserIDAsString: true,
	}
	return webauthn.New(cfg)
}

func rpIDAndOrigin(host string) (rpID, origin string, err error) {
	host = strings.TrimSpace(host)
	if host == "" {
		return "", "", fmt.Errorf("site host not configured")
	}
	if !strings.Contains(host, "://") {
		host = "https://" + host
	}
	u, err := url.Parse(host)
	if err != nil {
		return "", "", err
	}
	if u.Hostname() == "" {
		return "", "", fmt.Errorf("invalid site host")
	}
	rpID = u.Hostname()
	origin = u.Scheme + "://" + u.Host
	return rpID, origin, nil
}
