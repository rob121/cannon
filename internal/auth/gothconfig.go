package auth

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

// ConfigField describes one Goth provider setting for the admin form.
type ConfigField struct {
	Key      string
	Label    string
	Help     string
	Value    string
	Secret   bool
	Readonly bool
}

type fieldSpec struct {
	key      string
	label    string
	help     string
	secret   bool
	readonly bool
}

var (
	oauthClientKey = fieldSpec{key: "client_key", label: "Client Key", help: "OAuth client ID / application key (goth ClientKey)."}
	oauthSecret    = fieldSpec{key: "secret", label: "Client Secret", help: "OAuth client secret.", secret: true}
	oauthCallback  = fieldSpec{key: "callback_url", label: "Callback URL", help: "Redirect URI registered with the provider (goth CallbackURL)."}
	oauthScopes    = fieldSpec{key: "scopes", label: "Scopes", help: "Optional OAuth scopes, space or comma separated."}

	selfHostedAuth    = fieldSpec{key: "auth_url", label: "Auth URL", help: "OAuth authorize endpoint for self-hosted instances (optional)."}
	selfHostedToken   = fieldSpec{key: "token_url", label: "Token URL", help: "OAuth token endpoint for self-hosted instances (optional)."}
	selfHostedProfile = fieldSpec{key: "profile_url", label: "Profile URL", help: "User profile API endpoint for self-hosted instances (optional)."}
)

// providerSchemas maps Cannon authenticator names to Goth provider configuration keys.
// Keys follow goth New(...) parameters and commonly set Provider struct fields.
var providerSchemas = map[string][]fieldSpec{
	"local": {
		{key: "type", label: "Type", help: "Built-in password authentication.", readonly: true},
	},

	"amazon":       oauthStandard(),
	"battlenet":    oauthStandard(),
	"bitbucket":    oauthStandard(),
	"box":          oauthStandard(),
	"digitalocean": oauthStandard(),
	"dropbox":      oauthStandard(),
	"fitbit":       oauthStandard(),
	"github":       oauthStandard(),
	"heroku":       oauthStandard(),
	"instagram":    oauthStandard(),
	"intercom":     oauthStandard(),
	"kakao":        oauthStandard(),
	"linkedin":     oauthStandard(),
	"line":         oauthStandard(),
	"meetup":       oauthStandard(),
	"naver":        oauthNoScopes(),
	"paypal":       oauthStandard(),
	"salesforce":   oauthStandard(),
	"seatalk":      oauthStandard(),
	"slack":        oauthStandard(),
	"soundcloud":   oauthStandard(),
	"spotify":      oauthStandard(),
	"strava":       oauthStandard(),
	"stripe":       oauthStandard(),
	"tiktok":       oauthStandard(),
	"twitch":       oauthStandard(),
	"typetalk":     oauthStandard(),
	"uber":         oauthStandard(),
	"yahoo":        oauthStandard(),
	"yammer":       oauthStandard(),
	"yandex":       oauthStandard(),
	"zoom":         oauthStandard(),

	"apple": {
		{key: "client_id", label: "Client ID", help: "Apple Services ID (goth clientId)."},
		oauthSecret,
		{key: "callback_url", label: "Redirect URL", help: "Return URL registered in Apple Developer (goth redirectURL)."},
		oauthScopes,
	},

	"auth0": append(oauthNoScopes(),
		fieldSpec{key: "domain", label: "Auth0 Domain", help: "Tenant domain, e.g. example.us.auth0.com (goth auth0Domain)."},
		oauthScopes,
	),

	"azureadv2": append(oauthNoScopes(),
		fieldSpec{key: "tenant", label: "Tenant", help: "Azure AD tenant: common, organizations, consumers, or a tenant ID."},
		oauthScopes,
	),

	"discord": append(oauthStandard(),
		fieldSpec{key: "permissions", label: "Bot Permissions", help: "Discord bot permissions when using the bot scope (goth SetPermissions)."},
	),

	"facebook": append(oauthStandard(),
		fieldSpec{key: "fields", label: "Profile Fields", help: "Optional comma-separated Graph API user fields (goth SetCustomFields)."},
	),

	"gitea":     oauthSelfHosted(),
	"gitlab":    oauthSelfHosted(),
	"mastodon":  oauthSelfHosted(),
	"nextcloud": oauthSelfHosted(),
	"google":    oauthStandard(),

	"lastfm":  oauthNoScopes(),
	"twitter": oauthNoScopes(),
	"xero":    oauthNoScopes(),

	"microsoftonline": append(oauthNoScopes(),
		fieldSpec{key: "tenant", label: "Tenant", help: "Optional Azure tenant ID; defaults to common when empty."},
		fieldSpec{key: "scopes", label: "Scopes", help: "Optional scopes; Goth defaults to openid offline_access user.read."},
	),

	"okta": {
		{key: "client_id", label: "Client ID", help: "Okta application client ID (goth clientID)."},
		oauthSecret,
		{key: "org_url", label: "Org URL", help: "Okta organization URL, e.g. https://dev-123456.okta.com (goth orgURL)."},
		oauthCallback,
		oauthScopes,
	},

	"openid": {
		oauthClientKey,
		oauthSecret,
		oauthCallback,
		{key: "discovery_url", label: "Discovery URL", help: "OpenID Connect discovery document URL (goth openIDAutoDiscoveryURL)."},
		oauthScopes,
	},

	"shopify": append(oauthStandard(),
		fieldSpec{key: "shop_name", label: "Shop Name", help: "Shopify shop subdomain without .myshopify.com (goth SetShopName)."},
	),

	"steam": {
		{key: "api_key", label: "API Key", help: "Steam Web API key (goth apiKey)."},
		oauthCallback,
	},

	"vk": append(oauthStandard(),
		fieldSpec{key: "api_version", label: "API Version", help: "VK API version string (goth version)."},
	),

	"wecom": {
		{key: "corp_id", label: "Corp ID", help: "WeCom corp ID (goth corpID)."},
		oauthSecret,
		{key: "agent_id", label: "Agent ID", help: "WeCom agent ID (goth agentID)."},
		oauthCallback,
	},
}

func oauthStandard() []fieldSpec {
	return []fieldSpec{oauthClientKey, oauthSecret, oauthCallback, oauthScopes}
}

func oauthNoScopes() []fieldSpec {
	return []fieldSpec{oauthClientKey, oauthSecret, oauthCallback}
}

func oauthSelfHosted() []fieldSpec {
	return append(oauthStandard(), selfHostedAuth, selfHostedToken, selfHostedProfile)
}

// Schema returns configuration field definitions for a provider name.
func Schema(providerName string) []fieldSpec {
	if fields, ok := providerSchemas[providerName]; ok {
		return fields
	}
	return oauthStandard()
}

// DefaultConfig returns empty default values for a provider's configuration keys.
func DefaultConfig(providerName string) map[string]string {
	fields := Schema(providerName)
	out := make(map[string]string, len(fields))
	for _, f := range fields {
		if providerName == "local" && f.key == "type" {
			out[f.key] = "bcrypt"
			continue
		}
		out[f.key] = ""
	}
	return out
}

// DefaultConfigJSON returns JSON default configuration for seeding.
func DefaultConfigJSON(providerName string) string {
	data, err := json.Marshal(DefaultConfig(providerName))
	if err != nil {
		return "{}"
	}
	return string(data)
}

// MergedConfig combines stored JSON with schema defaults (stored values win).
func MergedConfig(providerName, stored string) map[string]string {
	merged := DefaultConfig(providerName)
	stored = strings.TrimSpace(stored)
	if stored == "" {
		return merged
	}
	var raw map[string]any
	if err := json.Unmarshal([]byte(stored), &raw); err != nil {
		return merged
	}
	for k, v := range raw {
		if v == nil {
			continue
		}
		merged[k] = fmt.Sprint(v)
	}
	return merged
}

// ConfigFormFields builds admin form fields with merged values.
func ConfigFormFields(providerName, stored string) []ConfigField {
	specs := Schema(providerName)
	values := MergedConfig(providerName, stored)
	fields := make([]ConfigField, 0, len(specs))
	for _, spec := range specs {
		fields = append(fields, ConfigField{
			Key:      spec.key,
			Label:    spec.label,
			Help:     spec.help,
			Value:    values[spec.key],
			Secret:   spec.secret,
			Readonly: spec.readonly,
		})
	}
	return fields
}

// ConfigFromForm reads cfg_* form fields and serializes provider configuration JSON.
func ConfigFromForm(r *http.Request, providerName string) string {
	values := DefaultConfig(providerName)
	for _, spec := range Schema(providerName) {
		if spec.readonly {
			continue
		}
		values[spec.key] = strings.TrimSpace(r.FormValue("cfg_" + spec.key))
	}
	data, err := json.Marshal(values)
	if err != nil {
		return DefaultConfigJSON(providerName)
	}
	return string(data)
}

// NormalizedConfigJSON returns stored configuration merged with schema defaults as JSON.
func NormalizedConfigJSON(providerName, stored string) string {
	data, err := json.Marshal(MergedConfig(providerName, stored))
	if err != nil {
		return DefaultConfigJSON(providerName)
	}
	return string(data)
}
