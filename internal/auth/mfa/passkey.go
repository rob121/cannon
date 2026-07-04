package mfa

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/go-webauthn/webauthn/protocol"
	"github.com/go-webauthn/webauthn/webauthn"
	"github.com/rob121/cannon/internal/models"
	"github.com/rob121/cannon/internal/sites"
	"github.com/rob121/cannon/internal/user"
	"gorm.io/gorm"
)

// WebAuthnUser adapts a Cannon user for go-webauthn.
type WebAuthnUser struct {
	User        models.User
	Credentials []webauthn.Credential
}

func (u WebAuthnUser) WebAuthnID() []byte {
	return []byte(strconv.FormatUint(uint64(u.User.UserID), 10))
}

func (u WebAuthnUser) WebAuthnName() string {
	if u.User.Username != "" {
		return u.User.Username
	}
	return u.User.Email
}

func (u WebAuthnUser) WebAuthnDisplayName() string {
	return user.DisplayName(&u.User)
}

func (u WebAuthnUser) WebAuthnCredentials() []webauthn.Credential {
	return u.Credentials
}

// LoadWebAuthnUser loads a user and passkey credentials for WebAuthn ceremonies.
func LoadWebAuthnUser(ctx context.Context, userID uint) (WebAuthnUser, error) {
	db, err := sites.DB(ctx)
	if err != nil {
		return WebAuthnUser{}, err
	}
	var u models.User
	if err := db.First(&u, userID).Error; err != nil {
		return WebAuthnUser{}, err
	}
	creds, err := LoadCredentials(db, userID)
	if err != nil {
		return WebAuthnUser{}, err
	}
	return WebAuthnUser{User: u, Credentials: creds}, nil
}

// LoadCredentials returns stored WebAuthn credentials for a user.
func LoadCredentials(db *gorm.DB, userID uint) ([]webauthn.Credential, error) {
	var rows []models.UserPasskey
	if err := db.Where("user_id = ?", userID).Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]webauthn.Credential, 0, len(rows))
	for _, row := range rows {
		var cred webauthn.Credential
		if err := json.Unmarshal([]byte(row.CredentialJSON), &cred); err != nil {
			continue
		}
		out = append(out, cred)
	}
	return out, nil
}

// ListPasskeys returns passkey metadata for account security UI.
func ListPasskeys(ctx context.Context, userID uint) ([]models.UserPasskey, error) {
	db, err := sites.DB(ctx)
	if err != nil {
		return nil, err
	}
	var rows []models.UserPasskey
	err = db.Where("user_id = ?", userID).Order("created_at asc").Find(&rows).Error
	return rows, err
}

// SavePasskey persists a new passkey credential.
func SavePasskey(ctx context.Context, userID uint, name string, cred webauthn.Credential) error {
	db, err := sites.DB(ctx)
	if err != nil {
		return err
	}
	raw, err := json.Marshal(cred)
	if err != nil {
		return err
	}
	if name == "" {
		name = "Passkey"
	}
	row := models.UserPasskey{
		UserID:         userID,
		Name:           name,
		CredentialJSON: string(raw),
	}
	return db.Create(&row).Error
}

// DeletePasskey removes a passkey owned by the user.
func DeletePasskey(ctx context.Context, userID, passkeyID uint) error {
	db, err := sites.DB(ctx)
	if err != nil {
		return err
	}
	res := db.Where("user_id = ? AND passkey_id = ?", userID, passkeyID).Delete(&models.UserPasskey{})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

// UpdateCredentialSignCount updates stored credential after successful assertion.
func UpdateCredentialSignCount(ctx context.Context, userID uint, cred webauthn.Credential) error {
	db, err := sites.DB(ctx)
	if err != nil {
		return err
	}
	var rows []models.UserPasskey
	if err := db.Where("user_id = ?", userID).Find(&rows).Error; err != nil {
		return err
	}
	now := time.Now()
	for _, row := range rows {
		var stored webauthn.Credential
		if err := json.Unmarshal([]byte(row.CredentialJSON), &stored); err != nil {
			continue
		}
		if string(stored.ID) != string(cred.ID) {
			continue
		}
		raw, err := json.Marshal(cred)
		if err != nil {
			return err
		}
		row.CredentialJSON = string(raw)
		row.LastUsedAt = &now
		return db.Save(&row).Error
	}
	return fmt.Errorf("passkey credential not found")
}

// BeginRegistration starts passkey enrollment for a signed-in user.
func BeginRegistration(ctx context.Context, svc *user.Service, userID uint) (map[string]any, error) {
	wa, err := WebAuthnInstance(ctx)
	if err != nil {
		return nil, err
	}
	wUser, err := LoadWebAuthnUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	creation, session, err := wa.BeginRegistration(wUser,
		webauthn.WithAuthenticatorSelection(protocol.AuthenticatorSelection{
			RequireResidentKey: protocol.ResidentKeyRequired(),
			UserVerification:   protocol.VerificationRequired,
		}),
	)
	if err != nil {
		return nil, err
	}
	if err := SaveWebAuthnSession(svc, session); err != nil {
		return nil, err
	}
	return map[string]any{"publicKey": creation.Response}, nil
}

// FinishRegistration completes passkey enrollment.
func FinishRegistration(ctx context.Context, svc *user.Service, userID uint, name string, r *http.Request) error {
	wa, err := WebAuthnInstance(ctx)
	if err != nil {
		return err
	}
	session, err := LoadWebAuthnSession(svc)
	if err != nil {
		return err
	}
	defer ClearWebAuthnSession(svc)

	wUser, err := LoadWebAuthnUser(ctx, userID)
	if err != nil {
		return err
	}
	cred, err := wa.FinishRegistration(wUser, *session, r)
	if err != nil {
		return err
	}
	return SavePasskey(ctx, userID, name, *cred)
}

// BeginLoginAssertion starts MFA passkey assertion for a pending user.
func BeginLoginAssertion(ctx context.Context, svc *user.Service, userID uint) (map[string]any, error) {
	wa, err := WebAuthnInstance(ctx)
	if err != nil {
		return nil, err
	}
	wUser, err := LoadWebAuthnUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	if len(wUser.Credentials) == 0 {
		return nil, errors.New("no passkeys registered")
	}
	assertion, session, err := wa.BeginLogin(wUser)
	if err != nil {
		return nil, err
	}
	if err := SaveWebAuthnSession(svc, session); err != nil {
		return nil, err
	}
	return map[string]any{"publicKey": assertion.Response}, nil
}

// BeginDiscoverableLogin starts passwordless passkey sign-in.
func BeginDiscoverableLogin(ctx context.Context, svc *user.Service) (map[string]any, error) {
	wa, err := WebAuthnInstance(ctx)
	if err != nil {
		return nil, err
	}
	assertion, session, err := wa.BeginDiscoverableLogin()
	if err != nil {
		return nil, err
	}
	if err := SaveWebAuthnSession(svc, session); err != nil {
		return nil, err
	}
	return map[string]any{"publicKey": assertion.Response}, nil
}

// FinishLoginAssertion completes passkey MFA for a known user.
func FinishLoginAssertion(ctx context.Context, svc *user.Service, userID uint, r *http.Request) error {
	wa, err := WebAuthnInstance(ctx)
	if err != nil {
		return err
	}
	session, err := LoadWebAuthnSession(svc)
	if err != nil {
		return err
	}
	defer ClearWebAuthnSession(svc)

	wUser, err := LoadWebAuthnUser(ctx, userID)
	if err != nil {
		return err
	}
	cred, err := wa.FinishLogin(wUser, *session, r)
	if err != nil {
		return err
	}
	return UpdateCredentialSignCount(ctx, userID, *cred)
}

// FinishDiscoverableLogin completes passwordless passkey sign-in and returns the user id.
func FinishDiscoverableLogin(ctx context.Context, svc *user.Service, r *http.Request) (uint, error) {
	wa, err := WebAuthnInstance(ctx)
	if err != nil {
		return 0, err
	}
	session, err := LoadWebAuthnSession(svc)
	if err != nil {
		return 0, err
	}
	defer ClearWebAuthnSession(svc)

	var resolvedUserID uint
	handler := func(rawID, userHandle []byte) (webauthn.User, error) {
		idText := string(userHandle)
		userID64, err := strconv.ParseUint(idText, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid user handle")
		}
		resolvedUserID = uint(userID64)
		return LoadWebAuthnUser(ctx, uint(userID64))
	}

	cred, err := wa.FinishDiscoverableLogin(handler, *session, r)
	if err != nil {
		return 0, err
	}
	if resolvedUserID == 0 {
		return 0, fmt.Errorf("user not resolved")
	}
	if err := UpdateCredentialSignCount(ctx, resolvedUserID, *cred); err != nil {
		return 0, err
	}
	return resolvedUserID, nil
}
