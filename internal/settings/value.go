package settings

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/rob121/cannon/internal/cache"
)

const (
	SectionGeneral   = "general"
	SectionMail      = "mail"
	SectionMedia     = "media"
	SectionSEO       = "seo"
	SectionAnalytics = "analytics"
)

// GlobalBool reads a boolean value from a global configuration section.
func GlobalBool(ctx context.Context, section, key string) (bool, error) {
	return GlobalBoolDefault(ctx, section, key, false)
}

// GlobalBoolDefault reads a boolean value, using fallback when the key is unset.
func GlobalBoolDefault(ctx context.Context, section, key string, fallback bool) (bool, error) {
	data, err := NewStore().Load(ctx, ScopeGlobal, section)
	if err != nil {
		return fallback, err
	}
	v, ok := data[key]
	if !ok || v == nil {
		return fallback, nil
	}
	return Bool(data, key), nil
}

// GlobalString reads a string from global settings.
func GlobalString(ctx context.Context, section, key string) (string, error) {
	data, err := NewStore().Load(ctx, ScopeGlobal, section)
	if err != nil {
		return "", err
	}
	v, ok := data[key]
	if !ok || v == nil {
		return "", nil
	}
	if s, ok := v.(string); ok {
		return s, nil
	}
	return strings.TrimSpace(fmt.Sprint(v)), nil
}

// GlobalInt reads an integer from global settings.
func GlobalIntDefault(ctx context.Context, section, key string, fallback int) (int, error) {
	data, err := NewStore().Load(ctx, ScopeGlobal, section)
	if err != nil {
		return fallback, err
	}
	v, ok := data[key]
	if !ok || v == nil {
		return fallback, nil
	}
	switch n := v.(type) {
	case float64:
		return int(n), nil
	case int:
		return n, nil
	case string:
		if i, err := strconv.Atoi(strings.TrimSpace(n)); err == nil {
			return i, nil
		}
	}
	return fallback, nil
}

// GlobalStringSlice reads a string array from global settings.
func GlobalStringSlice(ctx context.Context, section, key string) ([]string, error) {
	data, err := NewStore().Load(ctx, ScopeGlobal, section)
	if err != nil {
		return nil, err
	}
	v, ok := data[key]
	if !ok || v == nil {
		return nil, nil
	}
	switch raw := v.(type) {
	case []string:
		return raw, nil
	case []any:
		out := make([]string, 0, len(raw))
		for _, item := range raw {
			if s, ok := item.(string); ok {
				out = append(out, strings.TrimSpace(s))
			}
		}
		return out, nil
	default:
		text := strings.TrimSpace(fmt.Sprint(v))
		if text == "" {
			return nil, nil
		}
		parts := strings.Split(text, ",")
		out := make([]string, 0, len(parts))
		for _, part := range parts {
			part = strings.TrimSpace(part)
			if part != "" {
				out = append(out, part)
			}
		}
		return out, nil
	}
}

// AllowLogin reports whether public sign-in is enabled.
func AllowLogin(ctx context.Context) (bool, error) {
	return GlobalBoolDefault(ctx, SectionGeneral, "allow_login", true)
}

// AllowSignup reports whether public account registration is enabled.
func AllowSignup(ctx context.Context) (bool, error) {
	return GlobalBoolDefault(ctx, SectionGeneral, "allow_signup", true)
}

// AllowMFATOTP reports whether users may enroll TOTP authenticators.
func AllowMFATOTP(ctx context.Context) (bool, error) {
	return GlobalBoolDefault(ctx, SectionGeneral, "allow_mfa_totp", false)
}

// AllowPasskeys reports whether users may register and use passkeys.
func AllowPasskeys(ctx context.Context) (bool, error) {
	return GlobalBoolDefault(ctx, SectionGeneral, "allow_passkeys", false)
}

// CaptchaEnabled reports whether captcha placeholders should be expanded.
func CaptchaEnabled(ctx context.Context) (bool, error) {
	return GlobalBoolDefault(ctx, SectionGeneral, "captcha_enabled", false)
}

// CaptchaActiveExtension returns the configured captcha extension name.
func CaptchaActiveExtension(ctx context.Context) (string, error) {
	return GlobalString(ctx, SectionGeneral, "captcha_active_extension")
}

// CaptchaSkipAuthenticated reports whether signed-in users skip captcha render and verify.
func CaptchaSkipAuthenticated(ctx context.Context) (bool, error) {
	return GlobalBoolDefault(ctx, SectionGeneral, "captcha_skip_authenticated", true)
}

// SiteOffline reports whether the public site should show a maintenance page.
func SiteOffline(ctx context.Context) (bool, error) {
	if offline, err := GlobalBool(ctx, SectionGeneral, "site_offline"); err != nil {
		return false, err
	} else if offline {
		return true, nil
	}
	return GlobalBoolDefault(ctx, SectionGeneral, "maintenance_mode", false)
}

// DefaultListLimit returns the admin table page size.
func DefaultListLimit(ctx context.Context) (int, error) {
	limit, err := GlobalIntDefault(ctx, SectionGeneral, "default_list_limit", 25)
	if err != nil {
		return 25, err
	}
	return NormalizeListLimit(limit), nil
}

// NormalizeListLimit clamps list limits to supported admin page sizes.
func NormalizeListLimit(limit int) int {
	for _, allowed := range []int{25, 50, 75, 100, 200, 500} {
		if limit == allowed {
			return allowed
		}
	}
	if limit < 25 {
		return 25
	}
	if limit > 500 {
		return 500
	}
	return 25
}

// LogLevel returns the configured application log level name.
func LogLevel(ctx context.Context) (string, error) {
	level, err := GlobalString(ctx, SectionGeneral, "log_level")
	if err != nil || strings.TrimSpace(level) == "" {
		return "info", err
	}
	return strings.ToLower(strings.TrimSpace(level)), nil
}

// SiteMetaDescription returns the global default meta description.
func SiteMetaDescription(ctx context.Context) (string, error) {
	return globalStringWithLegacy(ctx, "site_meta_description")
}

// SiteMetaKeywords returns the global default meta keywords.
func SiteMetaKeywords(ctx context.Context) (string, error) {
	return globalStringWithLegacy(ctx, "site_meta_keywords")
}

// SiteOGTitle returns the global default Open Graph title.
func SiteOGTitle(ctx context.Context) (string, error) {
	return globalStringWithLegacy(ctx, "site_og_title")
}

// SiteOGImage returns the global default Open Graph image URL.
func SiteOGImage(ctx context.Context) (string, error) {
	return globalStringWithLegacy(ctx, "site_og_image")
}

// SiteTwitterCard returns the configured Twitter card type.
func SiteTwitterCard(ctx context.Context) (string, error) {
	card, err := globalStringWithLegacy(ctx, "site_twitter_card")
	if err != nil || strings.TrimSpace(card) == "" {
		return "summary_large_image", err
	}
	return strings.TrimSpace(card), nil
}

// SiteTwitterSite returns the site Twitter @handle.
func SiteTwitterSite(ctx context.Context) (string, error) {
	return globalStringWithLegacy(ctx, "site_twitter_site")
}

// SiteTwitterCreator returns the default Twitter creator @handle.
func SiteTwitterCreator(ctx context.Context) (string, error) {
	return globalStringWithLegacy(ctx, "site_twitter_creator")
}

// SiteHeadExtra returns additional head markup configured by an administrator.
func SiteHeadExtra(ctx context.Context) (string, error) {
	return globalStringWithLegacy(ctx, "site_head_extra")
}

func globalStringWithLegacy(ctx context.Context, key string) (string, error) {
	value, err := GlobalString(ctx, SectionSEO, key)
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(value) != "" {
		return value, nil
	}
	return GlobalString(ctx, SectionGeneral, key)
}

// FrontendTheme returns the active public site theme id.
func FrontendTheme(ctx context.Context) (string, error) {
	theme, err := GlobalString(ctx, SectionGeneral, "frontend_theme")
	if err != nil || strings.TrimSpace(theme) == "" {
		return "default", err
	}
	return strings.TrimSpace(theme), nil
}

// AdminTheme returns the active admin UI theme id.
func AdminTheme(ctx context.Context) (string, error) {
	theme, err := GlobalString(ctx, SectionGeneral, "admin_theme")
	if err != nil || strings.TrimSpace(theme) == "" {
		return "admin", err
	}
	return strings.TrimSpace(theme), nil
}

// RobotsTXT returns custom robots.txt body or empty for default generation.
func RobotsTXT(ctx context.Context) (string, error) {
	return GlobalString(ctx, SectionSEO, "robots_txt")
}

// AllowAICrawlers reports whether AI crawlers are allowed in robots.txt.
func AllowAICrawlers(ctx context.Context) (bool, error) {
	return GlobalBoolDefault(ctx, SectionSEO, "allow_ai_crawlers", true)
}

// AnalyticsEnabled reports whether live visitor analytics is active.
func AnalyticsEnabled(ctx context.Context) (bool, error) {
	return GlobalBoolDefault(ctx, SectionAnalytics, "enabled", false)
}

// AnalyticsAuthenticatedOnly reports whether only signed-in users are tracked.
func AnalyticsAuthenticatedOnly(ctx context.Context) (bool, error) {
	return GlobalBoolDefault(ctx, SectionAnalytics, "track_authenticated_only", false)
}

// MemoryCacheEnabled reports whether domain-level in-memory caches are active.
func MemoryCacheEnabled(ctx context.Context) (bool, error) {
	return cache.Enabled(ctx), nil
}

// Bool coerces a configuration map value to bool.
func Bool(data map[string]any, key string) bool {
	v, ok := data[key]
	if !ok || v == nil {
		return false
	}
	switch b := v.(type) {
	case bool:
		return b
	case float64:
		return b != 0
	case string:
		return b == "true" || b == "1"
	default:
		return false
	}
}
