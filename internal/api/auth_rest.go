package api

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/rob121/cannon/internal/auth/mfa"
	"github.com/rob121/cannon/internal/captcha"
	ctrlauth "github.com/rob121/cannon/internal/controllers/auth"
	"github.com/rob121/cannon/internal/models"
	"github.com/rob121/cannon/internal/notifications"
	"github.com/rob121/cannon/internal/settings"
	"github.com/rob121/cannon/internal/sites"
	"github.com/rob121/cannon/internal/user"
)

type passwordRequest struct {
	CurrentPassword string `json:"current_password"`
	NewPassword     string `json:"new_password"`
	ConfirmPassword string `json:"confirm_password"`
}

func (h *Handler) authChangePassword(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID, ok := h.requireJWT(w, r)
	if !ok {
		return
	}
	var req passwordRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", "Invalid JSON body")
		return
	}
	if req.NewPassword != req.ConfirmPassword {
		writeError(w, http.StatusBadRequest, "bad_request", "New passwords do not match")
		return
	}
	db, err := sites.DB(ctx)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "server_error", err.Error())
		return
	}
	var u models.User
	if err := db.First(&u, userID).Error; err != nil {
		writeError(w, http.StatusUnauthorized, "unauthorized", "User not found")
		return
	}
	if user.HasLocalPassword(db, &u) && strings.TrimSpace(req.CurrentPassword) == "" {
		writeError(w, http.StatusBadRequest, "bad_request", "Enter your current password")
		return
	}
	if err := user.UpdatePassword(ctx, userID, req.CurrentPassword, req.NewPassword); err != nil {
		msg := err.Error()
		if errors.Is(err, user.ErrInvalidPassword) {
			msg = "Current password is incorrect."
		} else if errors.Is(err, user.ErrPasswordShort) {
			msg = "Password must be at least 8 characters."
		}
		writeError(w, http.StatusBadRequest, "bad_request", msg)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) authUploadAvatar(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID, ok := h.requireJWT(w, r)
	if !ok {
		return
	}
	site, err := sites.FromContext(ctx)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "server_error", err.Error())
		return
	}
	if err := r.ParseMultipartForm(16 << 20); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", "Invalid multipart form")
		return
	}
	file, header, err := r.FormFile("avatar")
	if err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", "Avatar file is required")
		return
	}
	defer file.Close()
	webPath, err := user.SaveAvatarUpload(ctx, site, userID, file, header)
	if err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}
	if err := user.UpdateAvatarURL(ctx, userID, webPath); err != nil {
		writeError(w, http.StatusInternalServerError, "server_error", err.Error())
		return
	}
	body, err := buildMeResponse(ctx, userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "server_error", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, body)
}

func (h *Handler) authDeleteAvatar(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID, ok := h.requireJWT(w, r)
	if !ok {
		return
	}
	site, err := sites.FromContext(ctx)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "server_error", err.Error())
		return
	}
	if err := user.ClearAvatar(ctx, site, userID); err != nil {
		writeError(w, http.StatusInternalServerError, "server_error", err.Error())
		return
	}
	body, err := buildMeResponse(ctx, userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "server_error", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, body)
}

func (h *Handler) authGetNotifications(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID, ok := h.requireJWT(w, r)
	if !ok {
		return
	}
	state, err := notifications.LoadUserProfileState(ctx, userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "server_error", err.Error())
		return
	}
	groups := make([]map[string]any, 0, len(state.Groups))
	for _, g := range state.Groups {
		groups = append(groups, map[string]any{"id": g.ID, "label": g.Label, "events": g.Events})
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"groups": groups, "checked": state.Checked, "role_defaults": state.RoleDefaults,
	})
}

type notificationsPutRequest struct {
	Events []string `json:"events"`
}

func (h *Handler) authPutNotifications(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID, ok := h.requireJWT(w, r)
	if !ok {
		return
	}
	var req notificationsPutRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", "Invalid JSON body")
		return
	}
	if err := notifications.SaveUserSubscriptions(ctx, userID, req.Events); err != nil {
		writeError(w, http.StatusInternalServerError, "server_error", err.Error())
		return
	}
	h.authGetNotifications(w, r)
}

func (h *Handler) authPasswordReset(w http.ResponseWriter, r *http.Request, parts []string) {
	if len(parts) == 0 {
		writeError(w, http.StatusNotFound, "not_found", "Unknown endpoint")
		return
	}
	switch parts[0] {
	case "request":
		if r.Method != http.MethodPost {
			methodNotAllowed(w)
			return
		}
		h.authPasswordResetRequest(w, r)
	case "confirm":
		if r.Method != http.MethodPost {
			methodNotAllowed(w)
			return
		}
		h.authPasswordResetConfirm(w, r)
	default:
		writeError(w, http.StatusNotFound, "not_found", "Unknown endpoint")
	}
}

type resetRequestBody struct {
	Email        string `json:"email"`
	CaptchaToken string `json:"captcha_token"`
}

func (h *Handler) authPasswordResetRequest(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	var req resetRequestBody
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", "Invalid JSON body")
		return
	}
	_ = captcha.VerifyJSON(ctx, r, captcha.CaptchaContextForm, req.CaptchaToken)
	email := strings.TrimSpace(req.Email)
	if email != "" {
		db, err := sites.DB(ctx)
		if err == nil {
			var u models.User
			if db.Where("email = ?", email).First(&u).Error == nil {
				token, err := ctrlauth.IssueResetToken(ctx, u.UserID)
				if err == nil {
					sendPasswordResetEmail(r, u, token)
				}
			}
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{"message": "If that email is registered, a reset link has been sent."})
}

type resetConfirmBody struct {
	Token           string `json:"token"`
	Password        string `json:"password"`
	ConfirmPassword string `json:"confirm_password"`
}

func (h *Handler) authPasswordResetConfirm(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	var req resetConfirmBody
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", "Invalid JSON body")
		return
	}
	if req.Password != req.ConfirmPassword {
		writeError(w, http.StatusBadRequest, "bad_request", "Passwords do not match")
		return
	}
	row, err := ctrlauth.ConsumeToken(ctx, ctrlauth.TokenReset, strings.TrimSpace(req.Token))
	if err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", "Invalid or expired reset token")
		return
	}
	if err := setUserPasswordHash(ctx, row.UserID, req.Password); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"message": "Password updated."})
}

func (h *Handler) authSecurity(w http.ResponseWriter, r *http.Request, parts []string) {
	if len(parts) < 2 || parts[0] != "totp" {
		writeError(w, http.StatusNotFound, "not_found", "Unknown security endpoint")
		return
	}
	switch parts[1] {
	case "begin":
		if r.Method != http.MethodPost {
			methodNotAllowed(w)
			return
		}
		h.authTOTPBegin(w, r)
	case "confirm":
		if r.Method != http.MethodPost {
			methodNotAllowed(w)
			return
		}
		h.authTOTPConfirm(w, r)
	case "disable":
		if r.Method != http.MethodPost {
			methodNotAllowed(w)
			return
		}
		h.authTOTPDisable(w, r)
	default:
		writeError(w, http.StatusNotFound, "not_found", "Unknown security endpoint")
	}
}

func (h *Handler) authTOTPBegin(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID, ok := h.requireJWT(w, r)
	if !ok {
		return
	}
	allowed, err := settings.AllowMFATOTP(ctx)
	if err != nil || !allowed {
		writeError(w, http.StatusForbidden, "forbidden", "TOTP is disabled on this site")
		return
	}
	secret, err := mfa.GenerateTOTPSecret()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "server_error", err.Error())
		return
	}
	token, err := IssuePendingToken(ctx, PendingTOTPSetup, userID, map[string]string{"secret": secret}, 15*time.Minute)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "server_error", err.Error())
		return
	}
	db, err := sites.DB(ctx)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "server_error", err.Error())
		return
	}
	var u models.User
	_ = db.First(&u, userID).Error
	prov, _ := mfa.TOTPProvisioning(ctx, user.DisplayName(&u), secret)
	writeJSON(w, http.StatusOK, map[string]any{
		"totp_setup_token": token, "secret": secret, "uri": prov.URI, "qr_png_data_uri": prov.QRPNGDataURI,
	})
}

type totpConfirmBody struct {
	TOTPSetupToken string `json:"totp_setup_token"`
	Code           string `json:"code"`
}

func (h *Handler) authTOTPConfirm(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID, ok := h.requireJWT(w, r)
	if !ok {
		return
	}
	var req totpConfirmBody
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", "Invalid JSON body")
		return
	}
	row, err := ConsumePendingToken(ctx, PendingTOTPSetup, strings.TrimSpace(req.TOTPSetupToken))
	if err != nil || row.UserID != userID {
		writeError(w, http.StatusBadRequest, "bad_request", "Invalid setup session")
		return
	}
	var payload map[string]string
	_ = PendingPayload(row, &payload)
	secret := payload["secret"]
	if !mfa.ValidateTOTPCode(secret, strings.TrimSpace(req.Code)) {
		writeError(w, http.StatusBadRequest, "bad_request", "Invalid authentication code")
		return
	}
	if err := mfa.EnableUserTOTP(ctx, userID, secret); err != nil {
		writeError(w, http.StatusInternalServerError, "server_error", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"enabled": true})
}

type totpDisableBody struct {
	Code string `json:"code"`
}

func (h *Handler) authTOTPDisable(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID, ok := h.requireJWT(w, r)
	if !ok {
		return
	}
	var req totpDisableBody
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", "Invalid JSON body")
		return
	}
	okCode, err := mfa.VerifyUserTOTP(ctx, userID, strings.TrimSpace(req.Code))
	if err != nil || !okCode {
		writeError(w, http.StatusBadRequest, "bad_request", "Invalid authentication code")
		return
	}
	if err := mfa.DisableUserTOTP(ctx, userID); err != nil {
		writeError(w, http.StatusInternalServerError, "server_error", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"disabled": true})
}

func (h *Handler) authPasskeyLogin(w http.ResponseWriter, r *http.Request, parts []string) {
	writeError(w, http.StatusNotImplemented, "not_implemented", "Passkey login requires POST /auth/passkey-login/begin with mfa_token; WebAuthn flow is available from the account security endpoints in a future release.")
	_ = parts
	_ = r
}

func (h *Handler) authVerify(w http.ResponseWriter, r *http.Request, parts []string) {
	if len(parts) == 0 {
		writeError(w, http.StatusNotFound, "not_found", "Unknown endpoint")
		return
	}
	switch parts[0] {
	case "resend":
		if r.Method != http.MethodPost {
			methodNotAllowed(w)
			return
		}
		h.authVerifyResend(w, r)
	default:
		if r.Method != http.MethodPost {
			methodNotAllowed(w)
			return
		}
		h.authVerifyConfirm(w, r, parts[0])
	}
}

type verifyBody struct {
	Token string `json:"token"`
}

func (h *Handler) authVerifyConfirm(w http.ResponseWriter, r *http.Request, token string) {
	ctx := r.Context()
	var req verifyBody
	if err := decodeJSON(r, &req); err == nil && strings.TrimSpace(req.Token) != "" {
		token = strings.TrimSpace(req.Token)
	}
	row, err := ctrlauth.ConsumeToken(ctx, ctrlauth.TokenVerify, token)
	if err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", "Invalid or expired verification token")
		return
	}
	db, err := sites.DB(ctx)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "server_error", err.Error())
		return
	}
	if err := db.Model(&models.User{}).Where("user_id = ?", row.UserID).Update("validated", true).Error; err != nil {
		writeError(w, http.StatusInternalServerError, "server_error", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"verified": true})
}

type verifyResendBody struct {
	Email string `json:"email"`
}

func (h *Handler) authVerifyResend(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	var req verifyResendBody
	_ = decodeJSON(r, &req)
	writeJSON(w, http.StatusOK, map[string]any{"message": "If that account is pending verification, a new email has been sent."})
	email := strings.TrimSpace(req.Email)
	if email == "" {
		return
	}
	db, err := sites.DB(ctx)
	if err != nil {
		return
	}
	var u models.User
	if db.Where("email = ? AND validated = ?", email, false).First(&u).Error != nil {
		return
	}
	token, err := ctrlauth.EnsureVerifyToken(ctx, u.UserID)
	if err != nil {
		return
	}
	sendVerificationEmail(r, u, token)
}

func methodNotAllowed(w http.ResponseWriter) {
	writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "Method not allowed")
}
