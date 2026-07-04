package auth

import (
	"errors"
	"net/http"
	"strings"

	"github.com/rob121/cannon/internal/content"
	"github.com/rob121/cannon/internal/controllers"
	"github.com/rob121/cannon/internal/csrf"
	"github.com/rob121/cannon/internal/models"
	"github.com/rob121/cannon/internal/notifications"
	"github.com/rob121/cannon/internal/routepath"
	"github.com/rob121/cannon/internal/sites"
	"github.com/rob121/cannon/internal/user"
)

func (c *Controller) profile(ctx *controllers.Context) controllers.Result {
	u, err := ctx.CurrentUser()
	if err != nil {
		return controllers.Error(http.StatusUnauthorized, "sign in required")
	}
	db, err := sites.DB(ctx.GoContext())
	if err != nil {
		return controllers.Error(http.StatusInternalServerError, err.Error())
	}

	profileFields := []models.ProfileField{}
	profileInputs := []models.ContentField{}
	profileValues := map[uint]string{}
	authorProfileName := ""
	if profileID, err := content.AuthorProfileID(ctx.GoContext()); err == nil && profileID > 0 {
		profileFields, _ = content.ActiveProfileFields(ctx.GoContext(), profileID)
		for _, field := range profileFields {
			if content.IsAvatarProfileField(field) {
				continue
			}
			profileInputs = append(profileInputs, content.ProfileFieldAsContentField(field))
		}
		profileValues, _ = content.ProfileUserFieldValues(db, u.UserID, profileFields)
		var profile models.Profile
		if db.First(&profile, profileID).Error == nil {
			authorProfileName = profile.Name
		}
	}

	if ctx.Request.Method == http.MethodPost {
		if err := ctx.User.ValidateCSRF(ctx.Request); err != nil {
			return controllers.Error(http.StatusForbidden, "invalid form token")
		}
		switch strings.TrimSpace(ctx.Request.FormValue("form")) {
		case "password":
			return c.profilePasswordPost(ctx, u, profileInputs, profileValues, authorProfileName)
		case "notifications":
			return c.profileNotificationsPost(ctx, u, profileInputs, profileValues, authorProfileName)
		default:
			return c.profileSavePost(ctx, u, profileFields, profileInputs, profileValues, authorProfileName)
		}
	}

	return renderProfilePage(ctx, u, profileInputs, profileValues, authorProfileName, profilePageFlags{
		Saved:                ctx.Request.URL.Query().Get("saved") == "1",
		PasswordSaved:        ctx.Request.URL.Query().Get("password_saved") == "1",
		NotificationsSaved:   ctx.Request.URL.Query().Get("notifications_saved") == "1",
	})
}

type profilePageFlags struct {
	Saved              bool
	PasswordSaved      bool
	NotificationsSaved bool
	Error              string
}

func (c *Controller) profileSavePost(ctx *controllers.Context, u *models.User, profileFields []models.ProfileField, profileInputs []models.ContentField, profileValues map[uint]string, authorProfileName string) controllers.Result {
	if err := ctx.Request.ParseMultipartForm(16 << 20); err != nil {
		return profileError(ctx, u, profileInputs, profileValues, authorProfileName, "Invalid form submission.")
	}
	db, err := sites.DB(ctx.GoContext())
	if err != nil {
		return profileError(ctx, u, profileInputs, profileValues, authorProfileName, err.Error())
	}

	u.GivenName = strings.TrimSpace(ctx.Request.FormValue("given_name"))
	u.FamilyName = strings.TrimSpace(ctx.Request.FormValue("family_name"))
	username := strings.TrimSpace(ctx.Request.FormValue("username"))
	email := strings.TrimSpace(ctx.Request.FormValue("email"))

	if err := user.UpdateProfileIdentity(ctx.GoContext(), u.UserID, username, email); err != nil {
		msg := err.Error()
		if errors.Is(err, user.ErrUsernameTaken) {
			msg = "That username is already in use."
		} else if errors.Is(err, user.ErrEmailTaken) {
			msg = "That email is already in use."
		}
		return profileError(ctx, u, profileInputs, profileValues, authorProfileName, msg)
	}
	u.Username = username
	u.Email = email

	if err := db.Model(u).Updates(map[string]any{
		"given_name":  u.GivenName,
		"family_name": u.FamilyName,
	}).Error; err != nil {
		return profileError(ctx, u, profileInputs, profileValues, authorProfileName, err.Error())
	}
	if ctx.Request.FormValue("remove_avatar") == "1" {
		if err := user.ClearAvatar(ctx.GoContext(), ctx.Site, u.UserID); err != nil {
			return profileError(ctx, u, profileInputs, profileValues, authorProfileName, err.Error())
		}
	} else if file, header, err := ctx.Request.FormFile("avatar"); err == nil && file != nil {
		defer file.Close()
		webPath, err := user.SaveAvatarUpload(ctx.GoContext(), ctx.Site, u.UserID, file, header)
		if err != nil {
			return profileError(ctx, u, profileInputs, profileValues, authorProfileName, err.Error())
		}
		if err := user.UpdateAvatarURL(ctx.GoContext(), u.UserID, webPath); err != nil {
			return profileError(ctx, u, profileInputs, profileValues, authorProfileName, err.Error())
		}
	}
	if len(profileFields) > 0 {
		if err := content.SaveProfileUserFieldValuesWithUploads(ctx.GoContext(), ctx.Site, u.UserID, profileFields, ctx.Request); err != nil {
			return profileError(ctx, u, profileInputs, profileValues, authorProfileName, err.Error())
		}
	}
	dest := routepath.Controller(ctx.GoContext(), "auth", ActionProfile) + "?saved=1"
	return controllers.Redirect(http.StatusSeeOther, dest)
}

func (c *Controller) profilePasswordPost(ctx *controllers.Context, u *models.User, profileInputs []models.ContentField, profileValues map[uint]string, authorProfileName string) controllers.Result {
	if err := ctx.Request.ParseForm(); err != nil {
		return profileError(ctx, u, profileInputs, profileValues, authorProfileName, "Invalid form submission.")
	}
	current := ctx.Request.FormValue("current_password")
	newPassword := ctx.Request.FormValue("new_password")
	confirm := ctx.Request.FormValue("confirm_password")
	if newPassword != confirm {
		return profileError(ctx, u, profileInputs, profileValues, authorProfileName, "New passwords do not match.")
	}
	db, err := sites.DB(ctx.GoContext())
	if err != nil {
		return profileError(ctx, u, profileInputs, profileValues, authorProfileName, err.Error())
	}
	if user.HasLocalPassword(db, u) && strings.TrimSpace(current) == "" {
		return profileError(ctx, u, profileInputs, profileValues, authorProfileName, "Enter your current password.")
	}
	if err := user.UpdatePassword(ctx.GoContext(), u.UserID, current, newPassword); err != nil {
		msg := err.Error()
		switch {
		case errors.Is(err, user.ErrInvalidPassword):
			msg = "Current password is incorrect."
		case errors.Is(err, user.ErrPasswordShort):
			msg = "Password must be at least 8 characters."
		}
		return profileError(ctx, u, profileInputs, profileValues, authorProfileName, msg)
	}
	dest := routepath.Controller(ctx.GoContext(), "auth", ActionProfile) + "?password_saved=1#profile-password"
	return controllers.Redirect(http.StatusSeeOther, dest)
}

func (c *Controller) profileNotificationsPost(ctx *controllers.Context, u *models.User, profileInputs []models.ContentField, profileValues map[uint]string, authorProfileName string) controllers.Result {
	if err := ctx.Request.ParseForm(); err != nil {
		return profileError(ctx, u, profileInputs, profileValues, authorProfileName, "Invalid form submission.")
	}
	if err := notifications.SaveUserSubscriptions(ctx.GoContext(), u.UserID, ctx.Request.Form["notification_events"]); err != nil {
		return profileError(ctx, u, profileInputs, profileValues, authorProfileName, err.Error())
	}
	dest := routepath.Controller(ctx.GoContext(), "auth", ActionProfile) + "?notifications_saved=1#profile-notifications"
	return controllers.Redirect(http.StatusSeeOther, dest)
}

func profileError(ctx *controllers.Context, u *models.User, profileInputs []models.ContentField, profileValues map[uint]string, authorProfileName, message string) controllers.Result {
	return renderProfilePage(ctx, u, profileInputs, profileValues, authorProfileName, profilePageFlags{Error: message})
}

func renderProfilePage(ctx *controllers.Context, u *models.User, profileInputs []models.ContentField, profileValues map[uint]string, authorProfileName string, flags profilePageFlags) controllers.Result {
	token, _ := ctx.User.EnsureCSRFToken()
	avatarURL, _ := content.ResolveUserAvatar(ctx.GoContext(), u.UserID)
	db, _ := sites.DB(ctx.GoContext())
	hasLocalPassword := user.HasLocalPassword(db, u)
	notificationState, _ := notifications.LoadUserProfileState(ctx.GoContext(), u.UserID)
	data := map[string]any{
		"Title":             "Account Profile",
		"CSRF":              csrf.HiddenField(token),
		"Account":           u,
		"AvatarURL":         avatarURL,
		"DisplayName":       user.DisplayName(u),
		"ProfileFields":     profileInputs,
		"ProfileValues":     profileValues,
		"AuthorProfileName": authorProfileName,
		"HasLocalPassword":  hasLocalPassword,
		"Saved":             flags.Saved,
		"PasswordSaved":     flags.PasswordSaved,
		"NotificationsSaved": flags.NotificationsSaved,
		"NotificationGroups": notificationState.Groups,
		"NotificationChecked": notificationState.Checked,
		"NotificationRoleDefaults": notificationState.RoleDefaults,
		"ResetPasswordURL":  routepath.Controller(ctx.GoContext(), "auth", ActionResetRequest),
	}
	for k, v := range accountSecurityData(ctx, u) {
		data[k] = v
	}
	if strings.TrimSpace(u.SSOAvatarURL) != "" && strings.TrimSpace(u.AvatarURL) == "" {
		data["SSOAvatarURL"] = strings.TrimSpace(u.SSOAvatarURL)
	}
	if flags.Error != "" {
		data["Error"] = flags.Error
	}
	return controllers.HTML("Account Profile", data)
}
