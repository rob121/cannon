package settings

import (
	"context"
	"fmt"
	"strconv"
	"strings"
)

const (
	SectionGeneral = "general"
	SectionMail    = "mail"
	SectionMedia   = "media"
	SectionSEO     = "seo"
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

// AllowLogin reports whether public sign-in is enabled.
func AllowLogin(ctx context.Context) (bool, error) {
	return GlobalBoolDefault(ctx, SectionGeneral, "allow_login", true)
}

// AllowSignup reports whether public account registration is enabled.
func AllowSignup(ctx context.Context) (bool, error) {
	return GlobalBoolDefault(ctx, SectionGeneral, "allow_signup", true)
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
	return GlobalString(ctx, SectionGeneral, "site_meta_description")
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
