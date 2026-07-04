package api

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/rob121/cannon/internal/auth/mfa"
	"github.com/rob121/cannon/internal/captcha"
	cms "github.com/rob121/cannon/internal/content"
	"github.com/rob121/cannon/internal/hooks"
	"github.com/rob121/cannon/internal/models"
	"github.com/rob121/cannon/internal/settings"
	"github.com/rob121/cannon/internal/sites"
	"github.com/rob121/cannon/internal/user"
	"gorm.io/gorm"
)

func (h *Handler) serveAuth(w http.ResponseWriter, r *http.Request, parts []string) {
	if len(parts) == 0 {
		writeError(w, http.StatusNotFound, "not_found", "Unknown auth endpoint")
		return
	}
	switch parts[0] {
	case "login":
		if r.Method != http.MethodPost {
			methodNotAllowed(w)
			return
		}
		h.authLogin(w, r)
	case "mfa":
		if r.Method != http.MethodPost {
			methodNotAllowed(w)
			return
		}
		h.authMFA(w, r)
	case "refresh":
		if r.Method != http.MethodPost {
			methodNotAllowed(w)
			return
		}
		h.authRefresh(w, r)
	case "me":
		switch r.Method {
		case http.MethodGet:
			h.authMe(w, r)
		case http.MethodPatch:
			h.authPatchMe(w, r)
		default:
			methodNotAllowed(w)
		}
	case "password":
		if r.Method != http.MethodPost {
			methodNotAllowed(w)
			return
		}
		h.authChangePassword(w, r)
	case "avatar":
		switch r.Method {
		case http.MethodPost:
			h.authUploadAvatar(w, r)
		case http.MethodDelete:
			h.authDeleteAvatar(w, r)
		default:
			methodNotAllowed(w)
		}
	case "notifications":
		switch r.Method {
		case http.MethodGet:
			h.authGetNotifications(w, r)
		case http.MethodPut:
			h.authPutNotifications(w, r)
		default:
			methodNotAllowed(w)
		}
	case "logout":
		if r.Method != http.MethodPost {
			methodNotAllowed(w)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	case "password-reset":
		h.authPasswordReset(w, r, parts[1:])
	case "security":
		h.authSecurity(w, r, parts[1:])
	case "passkey-login":
		h.authPasskeyLogin(w, r, parts[1:])
	case "verify":
		h.authVerify(w, r, parts[1:])
	default:
		writeError(w, http.StatusNotFound, "not_found", "Unknown auth endpoint")
	}
}

type loginRequest struct {
	Username     string `json:"username"`
	Password     string `json:"password"`
	CaptchaToken string `json:"captcha_token"`
}

func (h *Handler) authLogin(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	limit, _ := loginRateLimit(ctx)
	key := "login:" + clientIP(r)
	if ok, retry := AllowRate(ctx, key, limit); !ok {
		writeRateLimited(w, retry)
		return
	}
	allowed, err := settings.AllowLogin(ctx)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "server_error", err.Error())
		return
	}
	if !allowed {
		writeError(w, http.StatusForbidden, "forbidden", "Sign in is currently disabled on this site.")
		return
	}
	var req loginRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", "Invalid JSON body")
		return
	}
	if err := captcha.VerifyJSON(ctx, r, captcha.CaptchaContextLogin, req.CaptchaToken); err != nil {
		writeError(w, http.StatusBadRequest, "captcha_failed", captcha.UserFacingError(err))
		return
	}
	username := strings.TrimSpace(req.Username)
	if username == "" || req.Password == "" {
		writeError(w, http.StatusUnauthorized, "unauthorized", "Invalid username or password")
		return
	}
	loginArgs := map[string]any{"username": username, "context": "api"}
	if _, err := hooks.Fire(ctx, hooks.OnUserBeforeLogin, loginArgs); err != nil {
		if errors.Is(err, hooks.ErrAborted) {
			msg := hooks.StringArg(loginArgs, "error")
			if msg == "" {
				msg = "Sign in is not allowed."
			}
			writeError(w, http.StatusForbidden, "forbidden", msg)
			return
		}
		writeError(w, http.StatusInternalServerError, "server_error", err.Error())
		return
	}
	u, err := user.AuthenticateLocal(ctx, username, req.Password)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "unauthorized", "Invalid username or password")
		return
	}
	if !u.Validated {
		writeJSON(w, http.StatusForbidden, map[string]any{
			"verify_required":  true,
			"message":          "Check your email to verify your account.",
			"resend_available": true,
		})
		return
	}
	if u.Locked || u.Status != models.StatusActive {
		writeError(w, http.StatusForbidden, "forbidden", "This account is locked.")
		return
	}
	needsMFA, err := mfa.UserRequiresMFA(ctx, u.UserID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "server_error", err.Error())
		return
	}
	if needsMFA {
		mfaToken, err := IssuePendingToken(ctx, PendingMFALogin, u.UserID, nil, 10*time.Minute)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "server_error", err.Error())
			return
		}
		methods := []string{}
		totpAllowed, _ := settings.AllowMFATOTP(ctx)
		passkeysAllowed, _ := settings.AllowPasskeys(ctx)
		db, _ := sites.DB(ctx)
		totpOn, _ := mfa.TOTPEnabled(db, u.UserID)
		passkeyCount, _ := mfa.PasskeyCount(db, u.UserID)
		if totpAllowed && totpOn {
			methods = append(methods, "totp")
		}
		resp := map[string]any{
			"mfa_required": true,
			"mfa_token":    mfaToken,
			"methods":      methods,
		}
		if passkeysAllowed && passkeyCount > 0 {
			methods = append(methods, "passkey")
			resp["passkey_required"] = true
			resp["passkey_begin_url"] = "/api/v1/auth/passkey-login/begin"
			resp["passkey_finish_url"] = "/api/v1/auth/passkey-login/finish"
			resp["mfa_token"] = mfaToken
		}
		resp["methods"] = methods
		writeJSON(w, http.StatusOK, resp)
		return
	}
	h.issueLoginTokens(w, r, u)
}

type mfaRequest struct {
	MFAToken string `json:"mfa_token"`
	Code     string `json:"code"`
}

func (h *Handler) authMFA(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	var req mfaRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", "Invalid JSON body")
		return
	}
	row, err := ConsumePendingToken(ctx, PendingMFALogin, strings.TrimSpace(req.MFAToken))
	if err != nil {
		writeError(w, http.StatusUnauthorized, "unauthorized", "Invalid or expired MFA session")
		return
	}
	code := strings.TrimSpace(req.Code)
	if code == "" {
		writeError(w, http.StatusBadRequest, "bad_request", "Authentication code is required")
		return
	}
	ok, err := mfa.VerifyUserTOTP(ctx, row.UserID, code)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "server_error", err.Error())
		return
	}
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized", "Invalid authentication code")
		return
	}
	db, err := sites.DB(ctx)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "server_error", err.Error())
		return
	}
	var u models.User
	if err := db.First(&u, row.UserID).Error; err != nil {
		writeError(w, http.StatusUnauthorized, "unauthorized", "User not found")
		return
	}
	h.issueLoginTokens(w, r, &u)
}

func (h *Handler) issueLoginTokens(w http.ResponseWriter, r *http.Request, u *models.User) {
	ctx := r.Context()
	_ = user.EnsureRegisteredGroup(ctx, u.UserID)
	access, expiresIn, err := IssueAccessToken(ctx, u.UserID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "server_error", err.Error())
		return
	}
	refresh, _ := IssueRefreshToken(ctx, u.UserID)
	avatar, _ := cms.ResolveUserAvatar(ctx, u.UserID)
	_, _ = hooks.Fire(ctx, hooks.OnUserAfterLogin, map[string]any{
		"context": "api", "user_id": u.UserID, "username": u.Username, "email": u.Email,
	})
	resp := map[string]any{
		"access_token": access,
		"token_type":   "Bearer",
		"expires_in":   expiresIn,
		"user": map[string]any{
			"user_id": u.UserID, "username": u.Username, "email": u.Email,
			"given_name": u.GivenName, "family_name": u.FamilyName, "avatar_url": avatar,
		},
	}
	if refresh != "" {
		resp["refresh_token"] = refresh
	}
	writeJSON(w, http.StatusOK, resp)
}

type refreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

func (h *Handler) authRefresh(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	var req refreshRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", "Invalid JSON body")
		return
	}
	userID, err := RefreshUserID(ctx, strings.TrimSpace(req.RefreshToken))
	if err != nil {
		writeError(w, http.StatusUnauthorized, "unauthorized", "Invalid refresh token")
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
	access, expiresIn, err := IssueAccessToken(ctx, u.UserID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "server_error", err.Error())
		return
	}
	refresh, _ := IssueRefreshToken(ctx, u.UserID)
	writeJSON(w, http.StatusOK, map[string]any{
		"access_token": access, "token_type": "Bearer", "expires_in": expiresIn, "refresh_token": refresh,
	})
}

func (h *Handler) authMe(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.requireJWT(w, r)
	if !ok {
		return
	}
	body, err := buildMeResponse(r.Context(), userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "server_error", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, body)
}

func buildMeResponse(ctx context.Context, userID uint) (map[string]any, error) {
	db, err := sites.DB(ctx)
	if err != nil {
		return nil, err
	}
	var u models.User
	if err := db.First(&u, userID).Error; err != nil {
		return nil, err
	}
	avatar, _ := cms.ResolveUserAvatar(ctx, userID)
	groups, _ := groupNamesForUser(ctx, userID)
	customFields, _ := profileCustomFields(ctx, userID)
	return map[string]any{
		"user_id": u.UserID, "username": u.Username, "email": u.Email,
		"given_name": u.GivenName, "family_name": u.FamilyName, "avatar_url": avatar,
		"groups": groups, "has_local_password": user.HasLocalPassword(db, &u),
		"custom_fields": customFields,
	}, nil
}

func profileCustomFields(ctx context.Context, userID uint) ([]map[string]any, error) {
	profileID, err := cms.AuthorProfileID(ctx)
	if err != nil || profileID == 0 {
		return nil, err
	}
	fields, err := cms.ActiveProfileFields(ctx, profileID)
	if err != nil {
		return nil, err
	}
	db, err := sites.DB(ctx)
	if err != nil {
		return nil, err
	}
	values, err := cms.ProfileUserFieldValues(db, userID, fields)
	if err != nil {
		return nil, err
	}
	var out []map[string]any
	for _, f := range fields {
		if cms.IsAvatarProfileField(f) {
			continue
		}
		out = append(out, map[string]any{
			"field_id": f.ProfileFieldID, "name": f.Name, "type": f.Type,
			"value": values[f.ProfileFieldID],
		})
	}
	return out, nil
}

type patchMeRequest struct {
	GivenName    *string           `json:"given_name"`
	FamilyName   *string           `json:"family_name"`
	Username     *string           `json:"username"`
	Email        *string           `json:"email"`
	CustomFields map[string]string `json:"custom_fields"`
}

func (h *Handler) authPatchMe(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID, ok := h.requireJWT(w, r)
	if !ok {
		return
	}
	var req patchMeRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", "Invalid JSON body")
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
	if req.Username != nil || req.Email != nil {
		un := u.Username
		em := u.Email
		if req.Username != nil {
			un = strings.TrimSpace(*req.Username)
		}
		if req.Email != nil {
			em = strings.TrimSpace(*req.Email)
		}
		if err := user.UpdateProfileIdentity(ctx, userID, un, em); err != nil {
			code := "bad_request"
			if errors.Is(err, user.ErrUsernameTaken) || errors.Is(err, user.ErrEmailTaken) {
				code = "conflict"
				writeError(w, http.StatusConflict, code, err.Error())
				return
			}
			writeError(w, http.StatusBadRequest, code, err.Error())
			return
		}
	}
	updates := map[string]any{}
	if req.GivenName != nil {
		updates["given_name"] = strings.TrimSpace(*req.GivenName)
	}
	if req.FamilyName != nil {
		updates["family_name"] = strings.TrimSpace(*req.FamilyName)
	}
	if len(updates) > 0 {
		_ = db.Model(&u).Updates(updates).Error
	}
	if len(req.CustomFields) > 0 {
		profileID, _ := cms.AuthorProfileID(ctx)
		if profileID > 0 {
			fields, _ := cms.ActiveProfileFields(ctx, profileID)
			_ = saveProfileFieldsFromMap(ctx, userID, fields, req.CustomFields)
		}
	}
	body, err := buildMeResponse(ctx, userID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "server_error", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, body)
}

func saveProfileFieldsFromMap(ctx context.Context, userID uint, fields []models.ProfileField, values map[string]string) error {
	db, err := sites.DB(ctx)
	if err != nil {
		return err
	}
	for _, f := range fields {
		if cms.IsAvatarProfileField(f) {
			continue
		}
		val, ok := values[strconvFieldID(f.ProfileFieldID)]
		if !ok {
			continue
		}
		var row models.ProfileUserFieldValue
		err := db.Where("user_id = ? AND field_id = ?", userID, f.ProfileFieldID).First(&row).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			row = models.ProfileUserFieldValue{UserID: userID, FieldID: f.ProfileFieldID, Value: val}
			if err := db.Create(&row).Error; err != nil {
				return err
			}
			continue
		}
		row.Value = val
		if err := db.Save(&row).Error; err != nil {
			return err
		}
	}
	return nil
}

func strconvFieldID(id uint) string {
	return strconv.FormatUint(uint64(id), 10)
}
