package templateengine

import (
	"context"
	"strings"

	"github.com/rob121/cannon/internal/settings"
)

const sectionContent = "content"

var publicTemplateSettings = map[string]map[string]struct{}{
	settings.SectionGeneral: {
		"site_name":              {},
		"date_format":            {},
		"frontend_theme":         {},
		"allow_login":            {},
		"allow_signup":           {},
		"content_multilingual":   {},
		"content_default_locale": {},
		"content_locales":        {},
	},
	settings.SectionSEO: {
		"site_meta_description": {},
		"site_meta_keywords":    {},
		"site_og_title":         {},
		"site_og_image":         {},
		"site_twitter_card":     {},
		"site_twitter_site":     {},
		"site_twitter_creator":  {},
		"site_head_extra":       {},
	},
	sectionContent: {
		"allow_comments":      {},
		"require_login":       {},
		"allow_anonymous":     {},
		"auto_approve":        {},
		"show_title":          {},
		"show_author":         {},
		"show_category_name":  {},
		"show_published_date": {},
		"show_author_bio":     {},
		"show_comments":       {},
	},
	settings.SectionAnalytics: {
		"enabled": {},
	},
}

// PublicSetting reads an allowlisted global setting for frontend templates.
func PublicSetting(ctx context.Context, section, key string) string {
	section = strings.TrimSpace(section)
	key = strings.TrimSpace(key)
	keys, ok := publicTemplateSettings[section]
	if !ok {
		return ""
	}
	if _, ok := keys[key]; !ok {
		return ""
	}
	value, err := settings.GlobalString(ctx, section, key)
	if err != nil {
		return ""
	}
	return value
}

// PublicSettingBool reads an allowlisted boolean global setting for frontend templates.
func PublicSettingBool(ctx context.Context, section, key string) bool {
	section = strings.TrimSpace(section)
	key = strings.TrimSpace(key)
	keys, ok := publicTemplateSettings[section]
	if !ok {
		return false
	}
	if _, ok := keys[key]; !ok {
		return false
	}
	value, err := settings.GlobalBool(ctx, section, key)
	if err != nil {
		return false
	}
	return value
}

// PublicSettingSlice reads an allowlisted string slice global setting for frontend templates.
func PublicSettingSlice(ctx context.Context, section, key string) []string {
	section = strings.TrimSpace(section)
	key = strings.TrimSpace(key)
	keys, ok := publicTemplateSettings[section]
	if !ok {
		return nil
	}
	if _, ok := keys[key]; !ok {
		return nil
	}
	value, err := settings.GlobalStringSlice(ctx, section, key)
	if err != nil {
		return nil
	}
	return value
}

// PublicSettingKey reports whether a setting key is exposed to frontend templates.
func PublicSettingKey(section, key string) bool {
	keys, ok := publicTemplateSettings[strings.TrimSpace(section)]
	if !ok {
		return false
	}
	_, ok = keys[strings.TrimSpace(key)]
	return ok
}

// PublicSettingSections returns documented public setting sections for help text.
func PublicSettingSections() []string {
	out := make([]string, 0, len(publicTemplateSettings))
	for section := range publicTemplateSettings {
		out = append(out, section)
	}
	return out
}

// PublicSettingKeys returns allowlisted keys for a section.
func PublicSettingKeys(section string) []string {
	keys, ok := publicTemplateSettings[strings.TrimSpace(section)]
	if !ok {
		return nil
	}
	out := make([]string, 0, len(keys))
	for key := range keys {
		out = append(out, key)
	}
	return out
}
