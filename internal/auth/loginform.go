package auth

import (
	"context"
	"html/template"
	"net/http"
	"strings"

	"github.com/rob121/cannon/internal/controllers"
	"github.com/rob121/cannon/internal/csrf"
	"github.com/rob121/cannon/internal/routepath"
	"github.com/rob121/cannon/internal/settings"
	"github.com/rob121/cannon/internal/sites"
	"github.com/rob121/cannon/internal/user"
)

// LoginFormOptions configures a rendered login form block.
type LoginFormOptions struct {
	BlockID uint
	Title   string
	Error   string
}

// OAuthProviderView is one external sign-in button for templates.
type OAuthProviderView struct {
	Name  string
	Label string
	URL   string
}

// LoginFormData is passed to login form templates.
type LoginFormData struct {
	BlockID         uint
	Title           string
	Disabled        bool
	DisabledMessage string
	Authenticated   bool
	Username        string
	LogoutURL       string
	LocalEnabled    bool
	ShowResetLink   bool
	LoginAction     string
	ResetURL        string
	Return          string
	Error           string
	CSRF            template.HTML
	Providers       []OAuthProviderView
}

// BuildLoginFormData assembles login form view data for a request.
func BuildLoginFormData(ctx context.Context, r *http.Request, opts LoginFormOptions) (LoginFormData, error) {
	data := LoginFormData{
		BlockID:       opts.BlockID,
		Title:         strings.TrimSpace(opts.Title),
		ShowResetLink: true,
		LoginAction:   routepath.Controller(ctx, "auth", "login"),
		ResetURL:      routepath.Controller(ctx, "auth", "reset-request"),
		LogoutURL:     routepath.Controller(ctx, "auth", "logout"),
		Error:         strings.TrimSpace(opts.Error),
	}

	allowed, err := settings.AllowLogin(ctx)
	if err != nil {
		return data, err
	}
	if !allowed {
		data.Disabled = true
		data.DisabledMessage = "Sign in is currently disabled on this site."
		return data, nil
	}

	providers, err := LoginProviders(ctx)
	if err != nil {
		return data, err
	}
	data.LocalEnabled = LocalLoginEnabled(providers)
	for _, provider := range OAuthProviders(providers) {
		begin := OAuthBeginPath(provider.Name)
		if begin == "" {
			continue
		}
		data.Providers = append(data.Providers, OAuthProviderView{
			Name:  provider.Name,
			Label: provider.Label,
			URL:   begin,
		})
	}

	if data.Error == "" {
		data.Error = strings.TrimSpace(r.URL.Query().Get("login_error"))
	}

	site, err := sites.FromContext(ctx)
	if err == nil && r != nil {
		returnPath := controllers.ReturnParam(site, r, r.URL.Path)
		if encoded, err := controllers.EncodeReturn(site, returnPath); err == nil {
			data.Return = encoded
		}
		for i := range data.Providers {
			if data.Return != "" {
				if withReturn, err := controllers.AppendReturn(site, data.Providers[i].URL, returnPath); err == nil {
					data.Providers[i].URL = withReturn
				}
			}
		}
	}

	if svc, err := user.FromContext(ctx); err == nil {
		if token, err := svc.EnsureCSRFToken(); err == nil {
			data.CSRF = csrf.HiddenField(token)
		}
		if u, err := svc.Current(ctx); err == nil {
			data.Authenticated = true
			if u.Username != "" {
				data.Username = u.Username
			} else {
				data.Username = u.Email
			}
		}
	}

	if !data.Authenticated && !data.Disabled && !data.LocalEnabled && len(data.Providers) == 0 {
		data.Disabled = true
		data.DisabledMessage = "No sign-in methods are configured. Enable authenticators in the admin."
	}
	return data, nil
}
