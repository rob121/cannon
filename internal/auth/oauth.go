package auth

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/markbates/goth"
	"github.com/markbates/goth/providers/apple"
	"github.com/markbates/goth/providers/auth0"
	"github.com/markbates/goth/providers/azureadv2"
	"github.com/markbates/goth/providers/bitbucket"
	"github.com/markbates/goth/providers/discord"
	"github.com/markbates/goth/providers/facebook"
	"github.com/markbates/goth/providers/github"
	"github.com/markbates/goth/providers/gitlab"
	"github.com/markbates/goth/providers/google"
	"github.com/markbates/goth/providers/linkedin"
	"github.com/markbates/goth/providers/microsoftonline"
	"github.com/markbates/goth/providers/okta"
	openidconnect "github.com/markbates/goth/providers/openidConnect"
	"github.com/markbates/goth/providers/slack"
	"github.com/markbates/goth/providers/spotify"
	"github.com/markbates/goth/providers/twitch"
	"github.com/markbates/goth/providers/twitter"
	"github.com/rob121/cannon/internal/content"
	"github.com/rob121/cannon/internal/models"
	"github.com/rob121/cannon/internal/sites"
	"github.com/rob121/cannon/internal/user"
	"gorm.io/gorm"
)

const (
	oauthSessionKey = "oauth:session:"
	oauthStateKey   = "oauth:state:"
	oauthReturnKey  = "oauth:return"
)

// BeginOAuth starts an OAuth sign-in flow for a provider.
func BeginOAuth(ctx context.Context, r *http.Request, svc *user.Service, providerName string, returnPath string) (string, error) {
	authRow, ok, err := IsActiveProvider(ctx, providerName)
	if err != nil {
		return "", err
	}
	if !ok {
		return "", fmt.Errorf("provider not available")
	}
	callbackURL := oauthCallbackURL(r, providerName, authRow.Configuration)
	provider, err := buildGothProvider(providerName, authRow.Configuration, callbackURL)
	if err != nil {
		return "", err
	}
	state, err := randomState()
	if err != nil {
		return "", err
	}
	session, err := provider.BeginAuth(state)
	if err != nil {
		return "", err
	}
	authURL, err := session.GetAuthURL()
	if err != nil {
		return "", err
	}
	if svc == nil {
		return "", fmt.Errorf("session unavailable")
	}
	if err := svc.SetSessionValue(oauthSessionKey+providerName, session.Marshal()); err != nil {
		return "", err
	}
	if err := svc.SetSessionValue(oauthStateKey+providerName, state); err != nil {
		return "", err
	}
	if strings.TrimSpace(returnPath) != "" {
		_ = svc.SetSessionValue(oauthReturnKey, strings.TrimSpace(returnPath))
	} else {
		_ = svc.ClearSessionValue(oauthReturnKey)
	}
	return authURL, nil
}

// CompleteOAuth finishes an OAuth callback and returns the signed-in user.
func CompleteOAuth(ctx context.Context, r *http.Request, svc *user.Service, providerName string) (*models.User, string, error) {
	authRow, ok, err := IsActiveProvider(ctx, providerName)
	if err != nil {
		return nil, "", err
	}
	if !ok {
		return nil, "", fmt.Errorf("provider not available")
	}
	if svc == nil {
		return nil, "", fmt.Errorf("session unavailable")
	}
	expectedState, _ := svc.SessionValue(oauthStateKey + providerName)
	if expectedState == nil || strings.TrimSpace(fmt.Sprint(expectedState)) == "" {
		return nil, "", fmt.Errorf("oauth session expired")
	}
	if got := strings.TrimSpace(r.URL.Query().Get("state")); got == "" || got != strings.TrimSpace(fmt.Sprint(expectedState)) {
		return nil, "", fmt.Errorf("invalid oauth state")
	}
	rawSession, ok := svc.SessionValue(oauthSessionKey + providerName)
	if !ok || rawSession == nil {
		return nil, "", fmt.Errorf("oauth session expired")
	}
	callbackURL := oauthCallbackURL(r, providerName, authRow.Configuration)
	provider, err := buildGothProvider(providerName, authRow.Configuration, callbackURL)
	if err != nil {
		return nil, "", err
	}
	session, err := provider.UnmarshalSession(fmt.Sprint(rawSession))
	if err != nil {
		return nil, "", err
	}
	params := url.Values{}
	for key, vals := range r.URL.Query() {
		if len(vals) > 0 {
			params.Set(key, vals[0])
		}
	}
	if _, err := session.Authorize(provider, params); err != nil {
		return nil, "", err
	}
	gu, err := provider.FetchUser(session)
	if err != nil {
		return nil, "", err
	}
	gu.Provider = providerName

	u, err := findOrCreateOAuthUser(ctx, authRow, gu)
	if err != nil {
		return nil, "", err
	}
	if err := user.SyncSSOAvatar(ctx, u.UserID, gu.AvatarURL); err != nil {
		return nil, "", err
	}
	if err := content.SyncProfileAvatarFromSSO(ctx, u.UserID, gu.AvatarURL); err != nil {
		return nil, "", err
	}

	returnPath := ""
	if v, ok := svc.SessionValue(oauthReturnKey); ok && v != nil {
		returnPath = strings.TrimSpace(fmt.Sprint(v))
	}
	_ = svc.ClearSessionValue(oauthSessionKey + providerName)
	_ = svc.ClearSessionValue(oauthStateKey + providerName)
	_ = svc.ClearSessionValue(oauthReturnKey)
	return u, returnPath, nil
}

func oauthCallbackURL(r *http.Request, providerName, configuration string) string {
	cfg := MergedConfig(providerName, configuration)
	if u := strings.TrimSpace(cfg["callback_url"]); u != "" {
		return u
	}
	scheme := "https"
	if r.TLS == nil {
		if proto := strings.TrimSpace(r.Header.Get("X-Forwarded-Proto")); proto != "" {
			scheme = proto
		} else {
			scheme = "http"
		}
	}
	host := strings.TrimSpace(r.Host)
	return scheme + "://" + host + OAuthCallbackPath(providerName)
}

// OAuthCallbackPath returns the callback URL path for a provider.
func OAuthCallbackPath(providerName string) string {
	return OAuthBeginPath(providerName) + "/callback"
}

func buildGothProvider(name, configuration, callbackURL string) (goth.Provider, error) {
	name = strings.ToLower(strings.TrimSpace(name))
	cfg := MergedConfig(name, configuration)
	scopes := parseOAuthScopes(cfg["scopes"])
	switch name {
	case "google":
		if scopes == nil {
			scopes = []string{"email", "profile"}
		}
		return google.New(cfg["client_key"], cfg["secret"], callbackURL, scopes...), nil
	case "github":
		return github.New(cfg["client_key"], cfg["secret"], callbackURL, scopes...), nil
	case "facebook":
		return facebook.New(cfg["client_key"], cfg["secret"], callbackURL, scopes...), nil
	case "gitlab":
		p := gitlab.New(cfg["client_key"], cfg["secret"], callbackURL, scopes...)
		if u := strings.TrimSpace(cfg["auth_url"]); u != "" {
			gitlab.AuthURL = strings.TrimRight(u, "/") + "/oauth/authorize"
		}
		if u := strings.TrimSpace(cfg["token_url"]); u != "" {
			gitlab.TokenURL = u
		}
		if u := strings.TrimSpace(cfg["profile_url"]); u != "" {
			gitlab.ProfileURL = u
		}
		return p, nil
	case "linkedin":
		return linkedin.New(cfg["client_key"], cfg["secret"], callbackURL, scopes...), nil
	case "microsoftonline":
		return microsoftonline.New(cfg["client_key"], cfg["secret"], callbackURL, scopes...), nil
	case "azureadv2":
		tenant := azureadv2.TenantType(strings.TrimSpace(cfg["tenant"]))
		if tenant == "" {
			tenant = azureadv2.CommonTenant
		}
		return azureadv2.New(cfg["client_key"], cfg["secret"], callbackURL, azureadv2.ProviderOptions{Tenant: tenant}), nil
	case "discord":
		return discord.New(cfg["client_key"], cfg["secret"], callbackURL, scopes...), nil
	case "twitch":
		return twitch.New(cfg["client_key"], cfg["secret"], callbackURL, scopes...), nil
	case "twitter":
		return twitter.New(cfg["client_key"], cfg["secret"], callbackURL), nil
	case "slack":
		return slack.New(cfg["client_key"], cfg["secret"], callbackURL, scopes...), nil
	case "spotify":
		return spotify.New(cfg["client_key"], cfg["secret"], callbackURL, scopes...), nil
	case "bitbucket":
		return bitbucket.New(cfg["client_key"], cfg["secret"], callbackURL, scopes...), nil
	case "auth0":
		return auth0.New(cfg["client_key"], cfg["secret"], callbackURL, cfg["domain"], scopes...), nil
	case "okta":
		return okta.New(cfg["client_id"], cfg["secret"], cfg["org_url"], callbackURL, scopes...), nil
	case "openid":
		return openidconnect.New(cfg["client_key"], cfg["secret"], callbackURL, cfg["discovery_url"], scopes...)
	case "apple":
		return apple.New(cfg["client_id"], cfg["secret"], callbackURL, nil, scopes...), nil
	default:
		if cfg["client_key"] != "" && cfg["secret"] != "" && callbackURL != "" {
			return nil, fmt.Errorf("oauth provider %q is not supported yet", name)
		}
		return nil, fmt.Errorf("oauth provider %q is not configured", name)
	}
}

func parseOAuthScopes(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	parts := strings.FieldsFunc(raw, func(r rune) bool {
		return r == ',' || r == ' ' || r == '\t' || r == '\n'
	})
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}

func randomState() (string, error) {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}

func findOrCreateOAuthUser(ctx context.Context, authRow models.Authenticator, gu goth.User) (*models.User, error) {
	db, err := sites.DB(ctx)
	if err != nil {
		return nil, err
	}
	providerUserID := strings.TrimSpace(gu.UserID)
	if providerUserID == "" {
		return nil, fmt.Errorf("provider did not return a user id")
	}
	var identity models.UserOAuthIdentity
	err = db.Where("provider = ? AND provider_user_id = ?", authRow.Name, providerUserID).First(&identity).Error
	if err == nil {
		var u models.User
		if err := db.First(&u, identity.UserID).Error; err != nil {
			return nil, err
		}
		return &u, updateOAuthUserProfile(db, &u, gu)
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	email := strings.TrimSpace(strings.ToLower(gu.Email))
	if email != "" {
		var existing models.User
		if err := db.Where("email = ?", email).First(&existing).Error; err == nil {
			if err := linkOAuthIdentity(db, existing.UserID, authRow.Name, providerUserID); err != nil {
				return nil, err
			}
			return &existing, updateOAuthUserProfile(db, &existing, gu)
		} else if !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, err
		}
	}

	username, err := uniqueOAuthUsername(db, gu)
	if err != nil {
		return nil, err
	}
	authID := authRow.AuthID
	u := models.User{
		GivenName:  strings.TrimSpace(gu.FirstName),
		FamilyName: strings.TrimSpace(gu.LastName),
		Email:      email,
		Username:   username,
		Status:     models.StatusActive,
		Validated:  true,
		AuthID:     &authID,
	}
	if strings.TrimSpace(u.GivenName) == "" && strings.TrimSpace(u.FamilyName) == "" {
		if name := strings.TrimSpace(gu.Name); name != "" {
			parts := strings.Fields(name)
			if len(parts) > 0 {
				u.GivenName = parts[0]
			}
			if len(parts) > 1 {
				u.FamilyName = strings.Join(parts[1:], " ")
			}
		} else if nick := strings.TrimSpace(gu.NickName); nick != "" {
			u.GivenName = nick
		}
	}
	if err := db.Create(&u).Error; err != nil {
		return nil, err
	}
	for _, name := range []string{"public", "registered"} {
		var g models.Group
		if err := db.Where("name = ?", name).First(&g).Error; err == nil {
			_ = db.Model(&u).Association("Groups").Append(&g)
		}
	}
	if err := linkOAuthIdentity(db, u.UserID, authRow.Name, providerUserID); err != nil {
		return nil, err
	}
	return &u, updateOAuthUserProfile(db, &u, gu)
}

func linkOAuthIdentity(db *gorm.DB, userID uint, provider, providerUserID string) error {
	row := models.UserOAuthIdentity{
		UserID:         userID,
		Provider:       provider,
		ProviderUserID: providerUserID,
	}
	return db.Create(&row).Error
}

func updateOAuthUserProfile(db *gorm.DB, u *models.User, gu goth.User) error {
	changed := false
	if strings.TrimSpace(u.GivenName) == "" && strings.TrimSpace(gu.FirstName) != "" {
		u.GivenName = strings.TrimSpace(gu.FirstName)
		changed = true
	}
	if strings.TrimSpace(u.FamilyName) == "" && strings.TrimSpace(gu.LastName) != "" {
		u.FamilyName = strings.TrimSpace(gu.LastName)
		changed = true
	}
	if strings.TrimSpace(u.Email) == "" && strings.TrimSpace(gu.Email) != "" {
		u.Email = strings.TrimSpace(strings.ToLower(gu.Email))
		changed = true
	}
	if !changed {
		return nil
	}
	return db.Save(u).Error
}

func uniqueOAuthUsername(db *gorm.DB, gu goth.User) (string, error) {
	candidates := []string{
		strings.TrimSpace(gu.NickName),
	}
	if email := strings.TrimSpace(gu.Email); email != "" {
		if at := strings.Index(email, "@"); at > 0 {
			candidates = append(candidates, email[:at])
		}
	}
	candidates = append(candidates, "user-"+strings.TrimSpace(gu.UserID))
	for _, base := range candidates {
		base = sanitizeUsername(base)
		if base == "" {
			continue
		}
		name := base
		for i := 0; i < 20; i++ {
			var count int64
			if err := db.Model(&models.User{}).Where("username = ?", name).Count(&count).Error; err != nil {
				return "", err
			}
			if count == 0 {
				return name, nil
			}
			name = fmt.Sprintf("%s-%d", base, i+2)
		}
	}
	return "", fmt.Errorf("could not generate a unique username")
}

func sanitizeUsername(raw string) string {
	raw = strings.ToLower(strings.TrimSpace(raw))
	var b strings.Builder
	for _, r := range raw {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
		case r == '-' || r == '_' || r == '.':
			b.WriteRune(r)
		}
	}
	out := strings.Trim(b.String(), "-_.")
	if len(out) > 64 {
		out = out[:64]
	}
	return out
}
