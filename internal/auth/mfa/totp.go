package mfa

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"image/png"
	"net/url"
	"strings"

	"github.com/pquerna/otp"
	"github.com/pquerna/otp/totp"
	"github.com/rob121/cannon/internal/models"
	"github.com/rob121/cannon/internal/settings"
	"github.com/rob121/cannon/internal/sites"
	"gorm.io/gorm"
)

// GenerateTOTPSecret creates a new base32 secret for enrollment.
func GenerateTOTPSecret() (string, error) {
	key, err := totp.Generate(totp.GenerateOpts{
		Issuer:      "Cannon",
		AccountName: "user",
	})
	if err != nil {
		return "", err
	}
	return key.Secret(), nil
}

// TOTPProvision holds enrollment display data for authenticator apps.
type TOTPProvision struct {
	URI          string
	QRPNGDataURI string
}

func totpIssuer(ctx context.Context) (string, error) {
	issuer, _ := settings.GlobalString(ctx, settings.SectionGeneral, "site_name")
	if issuer == "" {
		site, err := sites.FromContext(ctx)
		if err != nil {
			return "", err
		}
		issuer = site.Name
	}
	if issuer == "" {
		issuer = "Cannon"
	}
	return issuer, nil
}

// TOTPProvisioning builds an otpauth:// URI and QR code image for enrollment.
func TOTPProvisioning(ctx context.Context, accountName, secret string) (TOTPProvision, error) {
	issuer, err := totpIssuer(ctx)
	if err != nil {
		return TOTPProvision{}, err
	}
	accountName = strings.TrimSpace(accountName)
	if accountName == "" {
		accountName = "user"
	}
	key, err := otp.NewKeyFromURL(fmt.Sprintf("otpauth://totp/%s:%s?secret=%s&issuer=%s&algorithm=SHA1&digits=6&period=30",
		url.PathEscape(issuer),
		url.PathEscape(accountName),
		secret,
		url.QueryEscape(issuer),
	))
	if err != nil {
		return TOTPProvision{}, err
	}
	img, err := key.Image(256, 256)
	if err != nil {
		return TOTPProvision{}, err
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return TOTPProvision{}, err
	}
	return TOTPProvision{
		URI:          key.URL(),
		QRPNGDataURI: "data:image/png;base64," + base64.StdEncoding.EncodeToString(buf.Bytes()),
	}, nil
}

// TOTPProvisioningURI builds an otpauth:// URI for QR codes.
func TOTPProvisioningURI(ctx context.Context, accountName, secret string) (string, error) {
	p, err := TOTPProvisioning(ctx, accountName, secret)
	if err != nil {
		return "", err
	}
	return p.URI, nil
}

// ValidateTOTPCode checks a 6-digit code against a secret.
func ValidateTOTPCode(secret, code string) bool {
	code = strings.TrimSpace(code)
	if code == "" {
		return false
	}
	return totp.Validate(code, secret)
}

// LoadUserTOTP returns the user's TOTP row when present.
func LoadUserTOTP(db *gorm.DB, userID uint) (*models.UserTOTP, error) {
	var row models.UserTOTP
	err := db.Where("user_id = ?", userID).First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &row, nil
}

// TOTPEnabled reports whether the user has active TOTP MFA.
func TOTPEnabled(db *gorm.DB, userID uint) (bool, error) {
	row, err := LoadUserTOTP(db, userID)
	if err != nil || row == nil {
		return false, err
	}
	return row.Enabled, nil
}

// EnableUserTOTP saves and enables TOTP for a user.
func EnableUserTOTP(ctx context.Context, userID uint, secret string) error {
	db, err := sites.DB(ctx)
	if err != nil {
		return err
	}
	row, err := LoadUserTOTP(db, userID)
	if err != nil {
		return err
	}
	if row == nil {
		row = &models.UserTOTP{UserID: userID}
	}
	row.Secret = secret
	row.Enabled = true
	return db.Save(row).Error
}

// DisableUserTOTP removes TOTP enrollment for a user.
func DisableUserTOTP(ctx context.Context, userID uint) error {
	db, err := sites.DB(ctx)
	if err != nil {
		return err
	}
	return db.Where("user_id = ?", userID).Delete(&models.UserTOTP{}).Error
}

// VerifyUserTOTP validates a code for an enrolled user.
func VerifyUserTOTP(ctx context.Context, userID uint, code string) (bool, error) {
	db, err := sites.DB(ctx)
	if err != nil {
		return false, err
	}
	row, err := LoadUserTOTP(db, userID)
	if err != nil || row == nil || !row.Enabled {
		return false, err
	}
	return ValidateTOTPCode(row.Secret, code), nil
}
