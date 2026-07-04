package auth

import (
	"errors"
	"net/http"
	"net/url"
	"strings"

	"github.com/rob121/cannon/internal/auth"
	"github.com/rob121/cannon/internal/auth/mfa"
	"github.com/rob121/cannon/internal/captcha"
	"github.com/rob121/cannon/internal/controllers"
	"github.com/rob121/cannon/internal/hooks"
	"github.com/rob121/cannon/internal/models"
	"github.com/rob121/cannon/internal/paths"
	"github.com/rob121/cannon/internal/routepath"
	"github.com/rob121/cannon/internal/settings"
	"github.com/rob121/cannon/internal/sites"
	"github.com/rob121/cannon/internal/user"
	"golang.org/x/crypto/bcrypt"
)

const ControllerID = "auth"

const (
	ActionLogin         = "login"
	ActionLogout        = "logout"
	ActionOAuth         = "oauth"
	ActionVerify        = "verify"
	ActionVerifyResend  = "verify-resend"
	ActionResetRequest  = "reset-request"
	ActionResetSubmit   = "reset-submit"
	ActionMFAChallenge  = "mfa-challenge"
	ActionSecurity      = "security"
	ActionSecurityTOTP  = "security-totp"
	ActionSecurityPasskey = "security-passkey"
	ActionPasskeyLogin    = "passkey-login"
	ActionProfile         = "profile"
)

type Controller struct{}

func New() *Controller { return &Controller{} }

func Definition() controllers.Definition {
	return controllers.Definition{
		ID:          ControllerID,
		Title:       "Authentication",
		Description: "Login, logout, account verification, and password reset.",
		Actions: []controllers.ActionDefinition{
			{ID: ActionLogin, Title: "Login", Methods: []string{http.MethodGet, http.MethodPost}, DefaultPath: paths.AuthLogin, RequireGuest: true, AllowUnverified: true},
			{ID: ActionLogout, Title: "Logout", Methods: []string{http.MethodGet, http.MethodPost}, DefaultPath: paths.AuthLogout, AllowUnverified: true},
			{ID: ActionOAuth, Title: "OAuth Sign In", Methods: []string{http.MethodGet}, DefaultPath: paths.AuthOAuth, RequireGuest: true, AllowUnverified: true},
			{ID: ActionVerify, Title: "Verify Account", Methods: []string{http.MethodGet}, DefaultPath: paths.AccountVerify, AllowUnverified: true},
			{ID: ActionVerifyResend, Title: "Verification Pending", Methods: []string{http.MethodGet}, DefaultPath: paths.AccountVerifyResend, AllowUnverified: true},
			{ID: ActionResetRequest, Title: "Reset Password Request", Methods: []string{http.MethodGet, http.MethodPost}, DefaultPath: paths.AccountResetRequest, AllowUnverified: true},
			{ID: ActionResetSubmit, Title: "Reset Password Submit", Methods: []string{http.MethodGet, http.MethodPost}, DefaultPath: paths.AccountResetSubmit, AllowUnverified: true},
			{ID: ActionMFAChallenge, Title: "MFA Challenge", Methods: []string{http.MethodGet, http.MethodPost}, DefaultPath: paths.AccountMFAChallenge, RequireGuest: true, AllowUnverified: true},
			{ID: ActionSecurity, Title: "Account Security", Methods: []string{http.MethodGet}, DefaultPath: paths.AccountSecurity, RequireAuth: true},
			{ID: ActionSecurityTOTP, Title: "TOTP Setup", Methods: []string{http.MethodPost}, DefaultPath: paths.AccountSecurityTOTP, RequireAuth: true},
			{ID: ActionSecurityPasskey, Title: "Passkey Setup", Methods: []string{http.MethodGet, http.MethodPost}, DefaultPath: paths.AccountSecurityPasskey, RequireAuth: true},
			{ID: ActionPasskeyLogin, Title: "Passkey Login", Methods: []string{http.MethodPost}, DefaultPath: paths.AuthPasskeyLogin, RequireGuest: true, AllowUnverified: true},
			{ID: ActionProfile, Title: "Account Profile", Methods: []string{http.MethodGet, http.MethodPost}, DefaultPath: paths.AccountProfile, RequireAuth: true},
		},
	}
}

func (c *Controller) Handle(ctx *controllers.Context, actionID string) controllers.Result {
	switch actionID {
	case ActionLogin:
		return c.login(ctx)
	case ActionLogout:
		return c.logout(ctx)
	case ActionOAuth:
		return c.oauth(ctx)
	case ActionVerify:
		return c.verify(ctx)
	case ActionVerifyResend:
		return c.verifyResend(ctx)
	case ActionResetRequest:
		return c.resetRequest(ctx)
	case ActionResetSubmit:
		return c.resetSubmit(ctx)
	case ActionMFAChallenge:
		return c.mfaChallenge(ctx)
	case ActionSecurity:
		return c.security(ctx)
	case ActionSecurityTOTP:
		return c.securityTOTP(ctx)
	case ActionSecurityPasskey:
		return c.securityPasskey(ctx)
	case ActionPasskeyLogin:
		return c.passkeyLogin(ctx)
	case ActionProfile:
		return c.profile(ctx)
	default:
		return controllers.Error(http.StatusNotFound, "unknown auth action")
	}
}

func (c *Controller) login(ctx *controllers.Context) controllers.Result {
	r := ctx.Request
	allowed, err := settings.AllowLogin(ctx.GoContext())
	if err != nil {
		return controllers.Error(http.StatusInternalServerError, err.Error())
	}

	if r.Method == http.MethodGet {
		return renderLoginPage(ctx, "", r.URL.Query().Get("verified"))
	}

	if !allowed {
		return renderLoginPage(ctx, "Sign in is currently disabled on this site.", "")
	}

	if err := r.ParseForm(); err != nil {
		return loginError(ctx, "Invalid form submission.")
	}
	if err := captcha.VerifySubmit(ctx.GoContext(), r, captcha.CaptchaContextLogin); err != nil {
		return loginError(ctx, captcha.UserFacingError(err))
	}
	username := strings.TrimSpace(r.FormValue("username"))
	password := r.FormValue("password")
	if username == "" || password == "" {
		return loginError(ctx, "Username and password are required.")
	}

	loginArgs := hooks.RequestArgs(r)
	loginArgs["username"] = username
	loginArgs["context"] = "frontend"
	if _, err := hooks.Fire(ctx.GoContext(), hooks.OnUserBeforeLogin, loginArgs); err != nil {
		if errors.Is(err, hooks.ErrAborted) {
			msg := hooks.StringArg(loginArgs, "error")
			if msg == "" {
				msg = "Sign in is not allowed."
			}
			return loginError(ctx, msg)
		}
		return controllers.Error(http.StatusInternalServerError, err.Error())
	}

	u, err := user.AuthenticateLocal(ctx.GoContext(), username, password)
	if err != nil {
		return loginError(ctx, "Invalid username or password.")
	}
	if !u.Validated {
		return controllers.Redirect(http.StatusSeeOther, "/account/verify/resend")
	}
	if u.Locked {
		return loginError(ctx, "This account is locked.")
	}

	dest := returnPathFromRequest(ctx, r)
	if encoded := strings.TrimSpace(r.FormValue("return")); encoded != "" {
		if path, err := controllers.DecodeReturn(ctx.Site, encoded); err == nil {
			dest = path
		}
	}

	needsMFA, err := mfa.UserRequiresMFA(ctx.GoContext(), u.UserID)
	if err != nil {
		return controllers.Error(http.StatusInternalServerError, err.Error())
	}
	if needsMFA {
		if err := mfa.SetPendingMFA(ctx.User, mfa.PendingMFA{
			UserID:  u.UserID,
			Context: "frontend",
			Return:  dest,
		}); err != nil {
			return controllers.Error(http.StatusInternalServerError, "could not start MFA session")
		}
		return controllers.Redirect(http.StatusSeeOther, routepath.Controller(ctx.GoContext(), "auth", ActionMFAChallenge))
	}

	if err := completeFrontendLogin(ctx, u.UserID, "frontend"); err != nil {
		return controllers.Error(http.StatusInternalServerError, "could not start session")
	}

	return controllers.Redirect(http.StatusSeeOther, dest)
}

func loginError(ctx *controllers.Context, message string) controllers.Result {
	r := ctx.Request
	encoded := strings.TrimSpace(r.FormValue("return"))
	if encoded == "" {
		encoded = strings.TrimSpace(r.URL.Query().Get("return"))
	}
	if encoded != "" {
		if path, err := controllers.DecodeReturn(ctx.Site, encoded); err == nil {
			return controllers.Redirect(http.StatusSeeOther, appendQueryParam(path, "login_error", message))
		}
	}
	return renderLoginPage(ctx, message, r.URL.Query().Get("verified"))
}

func renderLoginPage(ctx *controllers.Context, message, verified string) controllers.Result {
	data, err := auth.BuildLoginFormData(ctx.GoContext(), ctx.Request, auth.LoginFormOptions{Error: message})
	if err != nil {
		return controllers.Error(http.StatusInternalServerError, err.Error())
	}
	return controllers.HTML("Sign In", map[string]any{
		"Login":    data,
		"Verified": verified,
	})
}

func (c *Controller) oauth(ctx *controllers.Context) controllers.Result {
	suffix := strings.Trim(strings.Trim(ctx.PathSuffix(), "/"), "/")
	parts := strings.Split(suffix, "/")
	providerName := strings.TrimSpace(parts[0])
	if providerName == "" {
		return controllers.Error(http.StatusNotFound, "unknown oauth provider")
	}
	isCallback := len(parts) > 1 && parts[1] == "callback"
	if isCallback {
		return c.oauthCallback(ctx, providerName)
	}
	if len(parts) > 1 {
		return controllers.Error(http.StatusNotFound, "unknown oauth path")
	}
	allowed, err := settings.AllowLogin(ctx.GoContext())
	if err != nil {
		return controllers.Error(http.StatusInternalServerError, err.Error())
	}
	if !allowed {
		return loginError(ctx, "Sign in is currently disabled on this site.")
	}
	returnPath := returnPathFromRequest(ctx, ctx.Request)
	authURL, err := auth.BeginOAuth(ctx.GoContext(), ctx.Request, ctx.User, providerName, returnPath)
	if err != nil {
		return loginError(ctx, "Could not start sign in with "+auth.ProviderDisplayName(providerName)+".")
	}
	return controllers.Redirect(http.StatusTemporaryRedirect, authURL)
}

func (c *Controller) oauthCallback(ctx *controllers.Context, providerName string) controllers.Result {
	u, returnPath, err := auth.CompleteOAuth(ctx.GoContext(), ctx.Request, ctx.User, providerName)
	if err != nil {
		return loginError(ctx, "Sign in with "+auth.ProviderDisplayName(providerName)+" failed. Try again or use local sign-in.")
	}
	if u.Locked {
		return loginError(ctx, "This account is locked.")
	}
	if !u.Validated {
		return controllers.Redirect(http.StatusSeeOther, "/account/verify/resend")
	}
	dest := strings.TrimSpace(returnPath)
	if dest == "" {
		dest = sites.DefaultRoutePath(ctx.GoContext())
	}
	needsMFA, err := mfa.UserRequiresMFA(ctx.GoContext(), u.UserID)
	if err != nil {
		return controllers.Error(http.StatusInternalServerError, err.Error())
	}
	if needsMFA {
		if err := mfa.SetPendingMFA(ctx.User, mfa.PendingMFA{
			UserID:  u.UserID,
			Context: "frontend",
			Return:  dest,
		}); err != nil {
			return controllers.Error(http.StatusInternalServerError, "could not start MFA session")
		}
		return controllers.Redirect(http.StatusSeeOther, routepath.Controller(ctx.GoContext(), "auth", ActionMFAChallenge))
	}
	if err := completeFrontendLogin(ctx, u.UserID, "frontend"); err != nil {
		return controllers.Error(http.StatusInternalServerError, "could not start session")
	}
	return controllers.Redirect(http.StatusSeeOther, dest)
}

func appendQueryParam(path, key, value string) string {
	sep := "?"
	if strings.Contains(path, "?") {
		sep = "&"
	}
	return path + sep + key + "=" + url.QueryEscape(value)
}

func returnPathFromRequest(ctx *controllers.Context, r *http.Request) string {
	if encoded := strings.TrimSpace(r.URL.Query().Get("return")); encoded != "" {
		if path, err := controllers.DecodeReturn(ctx.Site, encoded); err == nil {
			return path
		}
	}
	return sites.DefaultRoutePath(ctx.GoContext())
}

func (c *Controller) logout(ctx *controllers.Context) controllers.Result {
	if ctx.User != nil {
		if ctx.Authenticated() {
			if u, err := ctx.CurrentUser(); err == nil {
				logoutArgs := map[string]any{
					"context":  "frontend",
					"user_id":  u.UserID,
					"username": u.Username,
					"email":    u.Email,
				}
				if _, err := hooks.Fire(ctx.GoContext(), hooks.OnUserLogout, logoutArgs); err != nil {
					return controllers.Error(http.StatusInternalServerError, err.Error())
				}
			}
		}
		_ = ctx.User.Logout()
	}
	dest := routepath.Controller(ctx.GoContext(), "auth", "login")
	if allowed, err := settings.AllowLogin(ctx.GoContext()); err == nil && !allowed {
		dest = sites.DefaultRoutePath(ctx.GoContext())
	}
	return controllers.Redirect(http.StatusSeeOther, dest)
}

func (c *Controller) verify(ctx *controllers.Context) controllers.Result {
	token := strings.TrimSpace(ctx.PathSuffix())
	if token == "" {
		token = strings.TrimSpace(ctx.Request.URL.Query().Get("token"))
	}
	if token == "" {
		return controllers.HTML("Verify Account", map[string]any{"Error": "Verification link is invalid."})
	}
	row, err := ConsumeToken(ctx.GoContext(), TokenVerify, token)
	if err != nil {
		return controllers.HTML("Verify Account", map[string]any{"Error": "Verification link is invalid or expired."})
	}
	db, err := sites.DB(ctx.GoContext())
	if err != nil {
		return controllers.Error(http.StatusInternalServerError, err.Error())
	}
	if err := db.Model(&models.User{}).Where("user_id = ?", row.UserID).Update("validated", true).Error; err != nil {
		return controllers.Error(http.StatusInternalServerError, err.Error())
	}
	var u models.User
	if err := db.First(&u, row.UserID).Error; err == nil {
		_, _ = hooks.Fire(ctx.GoContext(), hooks.OnUserVerified, map[string]any{
			"user_id":  u.UserID,
			"username": u.Username,
			"email":    u.Email,
		})
	}
	return controllers.HTML("Account Verified", map[string]any{
		"Success": true,
		"Message": "Your account is verified. You may now sign in.",
	})
}

func (c *Controller) verifyResend(ctx *controllers.Context) controllers.Result {
	return controllers.HTML("Account Not Verified", map[string]any{
		"Message": "Your account must be verified before you can sign in. Contact a site administrator for a verification link.",
	})
}

func (c *Controller) resetRequest(ctx *controllers.Context) controllers.Result {
	r := ctx.Request
	if r.Method == http.MethodGet {
		return controllers.HTML("Reset Password", map[string]any{})
	}
	_ = r.ParseForm()
	identifier := strings.TrimSpace(r.FormValue("email"))
	if identifier == "" {
		identifier = strings.TrimSpace(r.FormValue("username"))
	}
	if identifier != "" {
		db, err := sites.DB(ctx.GoContext())
		if err == nil {
			var u models.User
			q := db.Where("status = ?", models.StatusActive)
			if strings.Contains(identifier, "@") {
				q = q.Where("email = ?", identifier)
			} else {
				q = q.Where("username = ?", identifier)
			}
			if err := q.First(&u).Error; err == nil && u.Validated {
				if token, err := IssueResetToken(ctx.GoContext(), u.UserID); err == nil {
					sendPasswordResetEmail(ctx, u, token)
				}
			}
		}
	}
	return controllers.HTML("Reset Password", map[string]any{
		"Sent":    true,
		"Message": "If a matching verified account exists, a reset link has been sent by email when mail is configured.",
	})
}

func (c *Controller) resetSubmit(ctx *controllers.Context) controllers.Result {
	r := ctx.Request
	token := strings.TrimSpace(ctx.PathSuffix())
	if token == "" {
		token = strings.TrimSpace(r.URL.Query().Get("token"))
	}
	if token == "" {
		return controllers.HTML("Reset Password", map[string]any{"Error": "Reset link is invalid."})
	}

	if r.Method == http.MethodGet {
		if _, err := LookupToken(ctx.GoContext(), TokenReset, token); err != nil {
			return controllers.HTML("Reset Password", map[string]any{"Error": "Reset link is invalid or expired."})
		}
		return controllers.HTML("Reset Password", map[string]any{"Token": token})
	}

	_ = r.ParseForm()
	password := r.FormValue("password")
	confirm := r.FormValue("password_confirm")
	if len(password) < 8 {
		return controllers.HTML("Reset Password", map[string]any{"Token": token, "Error": "Password must be at least 8 characters."})
	}
	if password != confirm {
		return controllers.HTML("Reset Password", map[string]any{"Token": token, "Error": "Passwords do not match."})
	}

	row, err := ConsumeToken(ctx.GoContext(), TokenReset, token)
	if err != nil {
		return controllers.HTML("Reset Password", map[string]any{"Error": "Reset link is invalid or expired."})
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return controllers.Error(http.StatusInternalServerError, "could not hash password")
	}
	db, err := sites.DB(ctx.GoContext())
	if err != nil {
		return controllers.Error(http.StatusInternalServerError, err.Error())
	}
	if err := db.Model(&models.User{}).Where("user_id = ?", row.UserID).Update("hash", string(hash)).Error; err != nil {
		return controllers.Error(http.StatusInternalServerError, err.Error())
	}
	return controllers.Redirect(http.StatusSeeOther, routepath.Controller(ctx.GoContext(), "auth", "login")+"?reset=1")
}
