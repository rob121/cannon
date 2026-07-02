package auth

import (
	"errors"
	"net/http"
	"strings"

	"github.com/rob121/cannon/internal/controllers"
	"github.com/rob121/cannon/internal/hooks"
	"github.com/rob121/cannon/internal/models"
	"github.com/rob121/cannon/internal/settings"
	"github.com/rob121/cannon/internal/sites"
	"github.com/rob121/cannon/internal/user"
	"golang.org/x/crypto/bcrypt"
)

const ControllerID = "auth"

const (
	ActionLogin         = "login"
	ActionLogout        = "logout"
	ActionVerify        = "verify"
	ActionVerifyResend  = "verify-resend"
	ActionResetRequest  = "reset-request"
	ActionResetSubmit   = "reset-submit"
)

type Controller struct{}

func New() *Controller { return &Controller{} }

func Definition() controllers.Definition {
	return controllers.Definition{
		ID:          ControllerID,
		Title:       "Authentication",
		Description: "Login, logout, account verification, and password reset.",
		Actions: []controllers.ActionDefinition{
			{ID: ActionLogin, Title: "Login", Methods: []string{http.MethodGet, http.MethodPost}, DefaultPath: "/login", RequireGuest: true, AllowUnverified: true},
			{ID: ActionLogout, Title: "Logout", Methods: []string{http.MethodGet, http.MethodPost}, DefaultPath: "/logout", AllowUnverified: true},
			{ID: ActionVerify, Title: "Verify Account", Methods: []string{http.MethodGet}, DefaultPath: "/account/verify/*", AllowUnverified: true},
			{ID: ActionVerifyResend, Title: "Verification Pending", Methods: []string{http.MethodGet}, DefaultPath: "/account/verify/resend", AllowUnverified: true},
			{ID: ActionResetRequest, Title: "Reset Password Request", Methods: []string{http.MethodGet, http.MethodPost}, DefaultPath: "/account/reset-password", AllowUnverified: true},
			{ID: ActionResetSubmit, Title: "Reset Password Submit", Methods: []string{http.MethodGet, http.MethodPost}, DefaultPath: "/account/reset-password/*", AllowUnverified: true},
		},
	}
}

func (c *Controller) Handle(ctx *controllers.Context, actionID string) controllers.Result {
	switch actionID {
	case ActionLogin:
		return c.login(ctx)
	case ActionLogout:
		return c.logout(ctx)
	case ActionVerify:
		return c.verify(ctx)
	case ActionVerifyResend:
		return c.verifyResend(ctx)
	case ActionResetRequest:
		return c.resetRequest(ctx)
	case ActionResetSubmit:
		return c.resetSubmit(ctx)
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
	if !allowed {
		return controllers.HTML("Sign In", map[string]any{
			"Disabled": true,
			"Message":  "Sign in is currently disabled on this site.",
		})
	}

	if r.Method == http.MethodGet {
		data := map[string]any{
			"Error":      r.URL.Query().Get("error"),
			"Verified":   r.URL.Query().Get("verified"),
			"Return":     r.URL.Query().Get("return"),
			"ReturnPath": returnPathFromRequest(ctx, r),
		}
		return controllers.HTML("Sign In", data)
	}

	if err := r.ParseForm(); err != nil {
		return loginError(ctx, "Invalid form submission.")
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
	if err := ctx.User.Login(u.UserID); err != nil {
		return controllers.Error(http.StatusInternalServerError, "could not start session")
	}
	afterArgs := map[string]any{
		"context":  "frontend",
		"user_id":  u.UserID,
		"username": u.Username,
		"email":    u.Email,
	}
	if _, err := hooks.Fire(ctx.GoContext(), hooks.OnUserAfterLogin, afterArgs); err != nil {
		return controllers.Error(http.StatusInternalServerError, err.Error())
	}

	dest := returnPathFromRequest(ctx, r)
	if encoded := strings.TrimSpace(r.FormValue("return")); encoded != "" {
		if path, err := controllers.DecodeReturn(ctx.Site, encoded); err == nil {
			dest = path
		}
	}
	return controllers.Redirect(http.StatusSeeOther, dest)
}

func loginError(ctx *controllers.Context, message string) controllers.Result {
	data := map[string]any{
		"Error":      message,
		"Return":     ctx.Request.URL.Query().Get("return"),
		"ReturnPath": returnPathFromRequest(ctx, ctx.Request),
	}
	return controllers.HTML("Sign In", data)
}

func returnPathFromRequest(ctx *controllers.Context, r *http.Request) string {
	if encoded := strings.TrimSpace(r.URL.Query().Get("return")); encoded != "" {
		if path, err := controllers.DecodeReturn(ctx.Site, encoded); err == nil {
			return path
		}
	}
	return "/"
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
	dest := "/login"
	if allowed, err := settings.AllowLogin(ctx.GoContext()); err == nil && !allowed {
		dest = "/"
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
				_, _ = IssueResetToken(ctx.GoContext(), u.UserID)
			}
		}
	}
	return controllers.HTML("Reset Password", map[string]any{
		"Sent":    true,
		"Message": "If a matching verified account exists, a reset link has been issued. Contact a site administrator to receive your reset link.",
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
	return controllers.Redirect(http.StatusSeeOther, "/login?reset=1")
}
