package auth

import (
	"context"
	"fmt"
	"html/template"
	"net/http"
	"strings"

	"github.com/rob121/cannon/internal/auth/mfa"
	"github.com/rob121/cannon/internal/controllers"
	"github.com/rob121/cannon/internal/models"
	"github.com/rob121/cannon/internal/routepath"
	"github.com/rob121/cannon/internal/settings"
	"github.com/rob121/cannon/internal/sites"
	"github.com/rob121/cannon/internal/user"
)

func (c *Controller) security(ctx *controllers.Context) controllers.Result {
	return controllers.Redirect(http.StatusSeeOther, profileSecurityRedirect(ctx.GoContext(), ctx.Request.URL.RawQuery))
}

func profileSecurityRedirect(ctx context.Context, rawQuery string) string {
	dest := routepath.Controller(ctx, "auth", ActionProfile)
	rawQuery = strings.TrimSpace(rawQuery)
	if rawQuery != "" {
		dest += "?" + rawQuery
	}
	return dest + "#account-security"
}

func accountSecurityData(ctx *controllers.Context, u *models.User) map[string]any {
	db, err := sites.DB(ctx.GoContext())
	if err != nil {
		return map[string]any{}
	}
	totpAllowed, _ := settings.AllowMFATOTP(ctx.GoContext())
	passkeysAllowed, _ := settings.AllowPasskeys(ctx.GoContext())
	totpEnabled, _ := mfa.TOTPEnabled(db, u.UserID)
	passkeys, _ := mfa.ListPasskeys(ctx.GoContext(), u.UserID)
	q := ctx.Request.URL.Query()
	data := map[string]any{
		"TOTPAllowed":         totpAllowed,
		"TOTPEnabled":         totpEnabled,
		"PasskeysAllowed":     passkeysAllowed,
		"Passkeys":            passkeys,
		"TOTPBeginURL":        routepath.ControllerWithSuffix(ctx.GoContext(), "auth", ActionSecurityTOTP, "begin"),
		"TOTPConfirmURL":      routepath.ControllerWithSuffix(ctx.GoContext(), "auth", ActionSecurityTOTP, "confirm"),
		"TOTPDisableURL":      routepath.ControllerWithSuffix(ctx.GoContext(), "auth", ActionSecurityTOTP, "disable"),
		"PasskeyBeginURL":     routepath.ControllerWithSuffix(ctx.GoContext(), "auth", ActionSecurityPasskey, "begin"),
		"PasskeyFinishURL":    routepath.ControllerWithSuffix(ctx.GoContext(), "auth", ActionSecurityPasskey, "finish"),
		"PasskeyDeleteURL":    routepath.ControllerWithSuffix(ctx.GoContext(), "auth", ActionSecurityPasskey, "delete"),
		"AccountName":         user.DisplayName(u),
		"FlashTOTPEnabled":    q.Get("totp_enabled") == "1",
		"FlashTOTPDisabled":   q.Get("totp_disabled") == "1",
		"FlashTOTPError":      q.Get("totp_error") == "1",
		"FlashPasskeyAdded":   q.Get("passkey_added") == "1",
		"FlashPasskeyRemoved": q.Get("passkey_removed") == "1",
	}
	if pending, ok := mfa.GetPendingTOTPSecret(ctx.User); ok {
		if prov, err := mfa.TOTPProvisioning(ctx.GoContext(), user.DisplayName(u), pending); err == nil {
			data["PendingTOTPSecret"] = pending
			data["PendingTOTPURI"] = template.URL(prov.URI)
			data["PendingTOTPQR"] = template.URL(prov.QRPNGDataURI)
		}
	}
	return data
}

func (c *Controller) securityTOTP(ctx *controllers.Context) controllers.Result {
	u, err := ctx.CurrentUser()
	if err != nil {
		return controllers.Error(http.StatusUnauthorized, "sign in required")
	}
	allowed, err := settings.AllowMFATOTP(ctx.GoContext())
	if err != nil {
		return controllers.Error(http.StatusInternalServerError, err.Error())
	}
	if !allowed {
		return controllers.Error(http.StatusForbidden, "TOTP is disabled on this site")
	}
	step := strings.Trim(strings.Trim(ctx.PathSuffix(), "/"), "/")
	r := ctx.Request
	switch step {
	case "begin":
		if r.Method != http.MethodPost {
			return controllers.Error(http.StatusMethodNotAllowed, "method not allowed")
		}
		if err := ctx.User.ValidateCSRF(r); err != nil {
			return controllers.Error(http.StatusForbidden, "invalid form token")
		}
		secret, err := mfa.GenerateTOTPSecret()
		if err != nil {
			return controllers.Error(http.StatusInternalServerError, err.Error())
		}
		if err := mfa.SetPendingTOTPSecret(ctx.User, secret); err != nil {
			return controllers.Error(http.StatusInternalServerError, err.Error())
		}
		return controllers.Redirect(http.StatusSeeOther, profileSecurityRedirect(ctx.GoContext(), ""))
	case "confirm":
		if r.Method != http.MethodPost {
			return controllers.Error(http.StatusMethodNotAllowed, "method not allowed")
		}
		if err := ctx.User.ValidateCSRF(r); err != nil {
			return controllers.Error(http.StatusForbidden, "invalid form token")
		}
		secret, ok := mfa.GetPendingTOTPSecret(ctx.User)
		if !ok {
			return controllers.Redirect(http.StatusSeeOther, profileSecurityRedirect(ctx.GoContext(), ""))
		}
		code := strings.TrimSpace(r.FormValue("code"))
		if !mfa.ValidateTOTPCode(secret, code) {
			return controllers.Redirect(http.StatusSeeOther, profileSecurityRedirect(ctx.GoContext(), "totp_error=1"))
		}
		if err := mfa.EnableUserTOTP(ctx.GoContext(), u.UserID, secret); err != nil {
			return controllers.Error(http.StatusInternalServerError, err.Error())
		}
		_ = mfa.ClearPendingTOTPSecret(ctx.User)
		return controllers.Redirect(http.StatusSeeOther, profileSecurityRedirect(ctx.GoContext(), "totp_enabled=1"))
	case "disable":
		if r.Method != http.MethodPost {
			return controllers.Error(http.StatusMethodNotAllowed, "method not allowed")
		}
		if err := ctx.User.ValidateCSRF(r); err != nil {
			return controllers.Error(http.StatusForbidden, "invalid form token")
		}
		code := strings.TrimSpace(r.FormValue("code"))
		ok, err := mfa.VerifyUserTOTP(ctx.GoContext(), u.UserID, code)
		if err != nil {
			return controllers.Error(http.StatusInternalServerError, err.Error())
		}
		if !ok {
			return controllers.Redirect(http.StatusSeeOther, profileSecurityRedirect(ctx.GoContext(), "totp_error=1"))
		}
		if err := mfa.DisableUserTOTP(ctx.GoContext(), u.UserID); err != nil {
			return controllers.Error(http.StatusInternalServerError, err.Error())
		}
		_ = mfa.ClearPendingTOTPSecret(ctx.User)
		return controllers.Redirect(http.StatusSeeOther, profileSecurityRedirect(ctx.GoContext(), "totp_disabled=1"))
	default:
		return controllers.Error(http.StatusNotFound, "unknown TOTP action")
	}
}

func (c *Controller) securityPasskey(ctx *controllers.Context) controllers.Result {
	u, err := ctx.CurrentUser()
	if err != nil {
		return controllers.Error(http.StatusUnauthorized, "sign in required")
	}
	allowed, err := settings.AllowPasskeys(ctx.GoContext())
	if err != nil {
		return controllers.Error(http.StatusInternalServerError, err.Error())
	}
	if !allowed {
		return controllers.Error(http.StatusForbidden, "passkeys are disabled on this site")
	}
	step := strings.Trim(strings.Trim(ctx.PathSuffix(), "/"), "/")
	r := ctx.Request
	switch step {
	case "begin":
		if r.Method != http.MethodPost {
			return controllers.Error(http.StatusMethodNotAllowed, "method not allowed")
		}
		payload, err := mfa.BeginRegistration(ctx.GoContext(), ctx.User, u.UserID)
		if err != nil {
			return controllers.JSON(http.StatusBadRequest, map[string]any{"error": err.Error()})
		}
		return controllers.JSON(http.StatusOK, payload)
	case "finish":
		if r.Method != http.MethodPost {
			return controllers.Error(http.StatusMethodNotAllowed, "method not allowed")
		}
		name := strings.TrimSpace(r.Header.Get("X-Passkey-Name"))
		if name == "" {
			name = strings.TrimSpace(r.FormValue("name"))
		}
		if name == "" {
			name = "Passkey"
		}
		if err := mfa.FinishRegistration(ctx.GoContext(), ctx.User, u.UserID, name, r); err != nil {
			return controllers.JSON(http.StatusBadRequest, map[string]any{"error": "passkey registration failed"})
		}
		return controllers.JSON(http.StatusOK, map[string]any{"redirect": profileSecurityRedirect(ctx.GoContext(), "passkey_added=1")})
	case "delete":
		if r.Method != http.MethodPost {
			return controllers.Error(http.StatusMethodNotAllowed, "method not allowed")
		}
		if err := ctx.User.ValidateCSRF(r); err != nil {
			return controllers.Error(http.StatusForbidden, "invalid form token")
		}
		idRaw := strings.TrimSpace(r.FormValue("passkey_id"))
		var passkeyID uint
		if _, err := fmt.Sscanf(idRaw, "%d", &passkeyID); err != nil || passkeyID == 0 {
			return controllers.Error(http.StatusBadRequest, "invalid passkey id")
		}
		if err := mfa.DeletePasskey(ctx.GoContext(), u.UserID, passkeyID); err != nil {
			return controllers.Error(http.StatusBadRequest, "passkey not found")
		}
		return controllers.Redirect(http.StatusSeeOther, profileSecurityRedirect(ctx.GoContext(), "passkey_removed=1"))
	default:
		return controllers.Error(http.StatusNotFound, "unknown passkey action")
	}
}
