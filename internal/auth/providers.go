package auth

import (
	"context"
	"strings"

	"github.com/rob121/cannon/internal/models"
)

// LoginProvider describes one sign-in method exposed to frontend forms.
type LoginProvider struct {
	Name  string
	Label string
	Kind  string // "local" or "oauth"
}

// ProviderDisplayName returns a human-readable authenticator label.
func ProviderDisplayName(name string) string {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "auth0":
		return "Auth0"
	case "azureadv2":
		return "Azure AD"
	case "openid":
		return "OpenID Connect"
	case "microsoftonline":
		return "Microsoft"
	case "wecom":
		return "WeCom"
	case "local":
		return "Local"
	case "":
		return ""
	default:
		name = strings.TrimSpace(name)
		return strings.ToUpper(name[:1]) + name[1:]
	}
}

// ProviderConfigured reports whether an authenticator has the credentials required to be offered.
func ProviderConfigured(name, configuration string) bool {
	name = strings.ToLower(strings.TrimSpace(name))
	if name == "" {
		return false
	}
	if name == "local" {
		return true
	}
	cfg := MergedConfig(name, configuration)
	switch name {
	case "apple":
		return cfg["client_id"] != "" && cfg["secret"] != "" && cfg["callback_url"] != ""
	case "auth0":
		return cfg["client_key"] != "" && cfg["secret"] != "" && cfg["callback_url"] != "" && cfg["domain"] != ""
	case "azureadv2", "microsoftonline":
		return cfg["client_key"] != "" && cfg["secret"] != "" && cfg["callback_url"] != ""
	case "okta":
		return cfg["client_id"] != "" && cfg["secret"] != "" && cfg["org_url"] != ""
	case "openid":
		return cfg["client_key"] != "" && cfg["secret"] != "" && cfg["callback_url"] != "" && cfg["discovery_url"] != ""
	case "steam":
		return cfg["api_key"] != "" && cfg["callback_url"] != ""
	case "vk":
		return cfg["client_key"] != "" && cfg["secret"] != "" && cfg["callback_url"] != "" && cfg["api_version"] != ""
	case "wecom":
		return cfg["corp_id"] != "" && cfg["secret"] != "" && cfg["agent_id"] != "" && cfg["callback_url"] != ""
	case "shopify":
		return cfg["client_key"] != "" && cfg["secret"] != "" && cfg["callback_url"] != "" && cfg["shop_name"] != ""
	case "gitea", "gitlab", "mastodon", "nextcloud":
		return cfg["client_key"] != "" && cfg["secret"] != "" && cfg["callback_url"] != "" && cfg["auth_url"] != "" && cfg["token_url"] != ""
	default:
		return cfg["client_key"] != "" && cfg["secret"] != "" && cfg["callback_url"] != ""
	}
}

// LoginProviders returns active, configured sign-in providers for frontend forms.
func LoginProviders(ctx context.Context) ([]LoginProvider, error) {
	rows, err := ActiveProviders(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]LoginProvider, 0, len(rows))
	for _, row := range rows {
		if !ProviderConfigured(row.Name, row.Configuration) {
			continue
		}
		kind := "oauth"
		if row.Name == "local" {
			kind = "local"
		}
		out = append(out, LoginProvider{
			Name:  row.Name,
			Label: ProviderDisplayName(row.Name),
			Kind:  kind,
		})
	}
	return out, nil
}

// LocalLoginEnabled reports whether password sign-in should be shown.
func LocalLoginEnabled(providers []LoginProvider) bool {
	for _, provider := range providers {
		if provider.Kind == "local" {
			return true
		}
	}
	return false
}

// OAuthProviders returns non-local providers from a provider list.
func OAuthProviders(providers []LoginProvider) []LoginProvider {
	out := make([]LoginProvider, 0)
	for _, provider := range providers {
		if provider.Kind != "local" {
			out = append(out, provider)
		}
	}
	return out
}

// OAuthBeginPath returns the relative begin URL for an OAuth provider.
func OAuthBeginPath(providerName string) string {
	name := strings.TrimSpace(providerName)
	if name == "" || name == "local" {
		return ""
	}
	return "/auth/oauth/" + name
}

// IsActiveProvider reports whether a named provider is active and configured.
func IsActiveProvider(ctx context.Context, name string) (models.Authenticator, bool, error) {
	rows, err := ActiveProviders(ctx)
	if err != nil {
		return models.Authenticator{}, false, err
	}
	for _, row := range rows {
		if row.Name == name && ProviderConfigured(row.Name, row.Configuration) {
			return row, true, nil
		}
	}
	return models.Authenticator{}, false, nil
}
