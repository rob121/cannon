package auth

import (
	"net/http"
	"strings"

	"github.com/rob121/cannon/internal/auth/mfa"
	"github.com/rob121/cannon/internal/controllers"
	"github.com/rob121/cannon/internal/csrf"
	"github.com/rob121/cannon/internal/routepath"
	"github.com/rob121/cannon/internal/settings"
	"github.com/rob121/cannon/internal/sites"
)

func (c *Controller) mfaChallenge(ctx *controllers.Context) controllers.Result {
	r := ctx.Request
	suffix := strings.Trim(strings.Trim(ctx.PathSuffix(), "/"), "/")
	if suffix == "passkey/begin" || suffix == "passkey/finish" {
		step := strings.TrimPrefix(suffix, "passkey/")
		return c.mfaChallengePasskey(ctx, step)
	}

	pending, ok := mfa.GetPendingMFA(ctx.User)
	if !ok {
		return controllers.Redirect(http.StatusSeeOther, routepath.Controller(ctx.GoContext(), "auth", ActionLogin))
	}

	if r.Method == http.MethodGet {
		return renderMFAChallengePage(ctx, pending, "")
	}

	if err := r.ParseForm(); err != nil {
		return renderMFAChallengePage(ctx, pending, "Invalid form submission.")
	}
	if err := ctx.User.ValidateCSRF(r); err != nil {
		return renderMFAChallengePage(ctx, pending, "Invalid or expired form token. Refresh and try again.")
	}

	code := strings.TrimSpace(r.FormValue("code"))
	if code != "" {
		ok, err := mfa.VerifyUserTOTP(ctx.GoContext(), pending.UserID, code)
		if err != nil {
			return controllers.Error(http.StatusInternalServerError, err.Error())
		}
		if !ok {
			return renderMFAChallengePage(ctx, pending, "Invalid authentication code.")
		}
		return finishPendingLogin(ctx, pending)
	}

	return renderMFAChallengePage(ctx, pending, "Enter your authentication code or use a passkey.")
}

func renderMFAChallengePage(ctx *controllers.Context, pending mfa.PendingMFA, message string) controllers.Result {
	totpAllowed, _ := settings.AllowMFATOTP(ctx.GoContext())
	passkeysAllowed, _ := settings.AllowPasskeys(ctx.GoContext())

	db, err := sites.DB(ctx.GoContext())
	if err != nil {
		return controllers.Error(http.StatusInternalServerError, err.Error())
	}
	totpEnabled, _ := mfa.TOTPEnabled(db, pending.UserID)
	passkeyCount, _ := mfa.PasskeyCount(db, pending.UserID)

	token, _ := ctx.User.EnsureCSRFToken()
	data := map[string]any{
		"Title":              "Verify Sign In",
		"Error":              message,
		"CSRF":               csrf.HiddenField(token),
		"Action":             routepath.Controller(ctx.GoContext(), "auth", ActionMFAChallenge),
		"TOTPAllowed":        totpAllowed && totpEnabled,
		"PasskeysAllowed":    passkeysAllowed && passkeyCount > 0,
		"PasskeyBeginURL":    routepath.ControllerWithSuffix(ctx.GoContext(), "auth", ActionMFAChallenge, "passkey/begin"),
		"PasskeyFinishURL":   routepath.ControllerWithSuffix(ctx.GoContext(), "auth", ActionMFAChallenge, "passkey/finish"),
	}
	return controllers.HTML("Verify Sign In", data)
}

func (c *Controller) mfaChallengePasskey(ctx *controllers.Context, step string) controllers.Result {
	pending, ok := mfa.GetPendingMFA(ctx.User)
	if !ok {
		return controllers.JSON(http.StatusUnauthorized, map[string]any{"error": "sign-in session expired"})
	}
	r := ctx.Request
	switch step {
	case "begin":
		if r.Method != http.MethodPost {
			return controllers.Error(http.StatusMethodNotAllowed, "method not allowed")
		}
		payload, err := mfa.BeginLoginAssertion(ctx.GoContext(), ctx.User, pending.UserID)
		if err != nil {
			return controllers.JSON(http.StatusBadRequest, map[string]any{"error": err.Error()})
		}
		return controllers.JSON(http.StatusOK, payload)
	case "finish":
		if r.Method != http.MethodPost {
			return controllers.Error(http.StatusMethodNotAllowed, "method not allowed")
		}
		if err := mfa.FinishLoginAssertion(ctx.GoContext(), ctx.User, pending.UserID, r); err != nil {
			return controllers.JSON(http.StatusUnauthorized, map[string]any{"error": "passkey verification failed"})
		}
		dest := pendingLoginDestination(ctx, pending)
		if err := completeFrontendLogin(ctx, pending.UserID, pending.Context); err != nil {
			return controllers.Error(http.StatusInternalServerError, err.Error())
		}
		_ = mfa.ClearPendingMFA(ctx.User)
		return controllers.JSON(http.StatusOK, map[string]any{"redirect": dest})
	default:
		return controllers.Error(http.StatusNotFound, "unknown passkey step")
	}
}

func (c *Controller) passkeyLogin(ctx *controllers.Context) controllers.Result {
	allowed, err := settings.AllowPasskeys(ctx.GoContext())
	if err != nil {
		return controllers.Error(http.StatusInternalServerError, err.Error())
	}
	if !allowed {
		return controllers.JSON(http.StatusForbidden, map[string]any{"error": "passkeys are disabled"})
	}
	step := strings.Trim(strings.Trim(ctx.PathSuffix(), "/"), "/")
	r := ctx.Request
	switch step {
	case "begin":
		if r.Method != http.MethodPost {
			return controllers.Error(http.StatusMethodNotAllowed, "method not allowed")
		}
		payload, err := mfa.BeginDiscoverableLogin(ctx.GoContext(), ctx.User)
		if err != nil {
			return controllers.JSON(http.StatusBadRequest, map[string]any{"error": err.Error()})
		}
		return controllers.JSON(http.StatusOK, payload)
	case "finish":
		if r.Method != http.MethodPost {
			return controllers.Error(http.StatusMethodNotAllowed, "method not allowed")
		}
		userID, err := mfa.FinishDiscoverableLogin(ctx.GoContext(), ctx.User, r)
		if err != nil {
			return controllers.JSON(http.StatusUnauthorized, map[string]any{"error": "passkey sign-in failed"})
		}
		u, err := loadUserByID(ctx, userID)
		if err != nil {
			return controllers.JSON(http.StatusUnauthorized, map[string]any{"error": "account not found"})
		}
		if !u.Validated || u.Locked {
			return controllers.JSON(http.StatusForbidden, map[string]any{"error": "account cannot sign in"})
		}
		if err := completeFrontendLogin(ctx, userID, "frontend"); err != nil {
			return controllers.Error(http.StatusInternalServerError, err.Error())
		}
		dest := returnPathFromRequest(ctx, r)
		if encoded := strings.TrimSpace(r.URL.Query().Get("return")); encoded != "" {
			if path, err := controllers.DecodeReturn(ctx.Site, encoded); err == nil {
				dest = path
			}
		}
		return controllers.JSON(http.StatusOK, map[string]any{"redirect": dest})
	default:
		return controllers.Error(http.StatusNotFound, "unknown passkey step")
	}
}

func finishPendingLogin(ctx *controllers.Context, pending mfa.PendingMFA) controllers.Result {
	dest := pendingLoginDestination(ctx, pending)
	if err := completeFrontendLogin(ctx, pending.UserID, pending.Context); err != nil {
		return controllers.Error(http.StatusInternalServerError, err.Error())
	}
	_ = mfa.ClearPendingMFA(ctx.User)
	return controllers.Redirect(http.StatusSeeOther, dest)
}

func pendingLoginDestination(ctx *controllers.Context, pending mfa.PendingMFA) string {
	if pending.Return != "" {
		return pending.Return
	}
	return returnPathFromRequest(ctx, ctx.Request)
}
