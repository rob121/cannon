package content

import (
	"context"

	"github.com/rob121/cannon/internal/settings"
)

const SettingsSection = "content"

// Settings holds global content configuration.
type Settings struct {
	AllowComments      bool
	RequireLogin       bool
	AutoApprove        bool
	AllowAnonymous     bool
	ShowTitle          bool
	ShowAuthor         bool
	ShowPublishedDate  bool
	ShowAuthorBio      bool
	ShowComments       bool
	AuthorProfileID    uint
}

// LoadSettings reads content settings from the global settings store.
func LoadSettings(ctx context.Context) (Settings, error) {
	store := settings.NewStore()
	data, err := store.Load(ctx, settings.ScopeGlobal, SettingsSection)
	if err != nil {
		return Settings{}, err
	}
	return Settings{
		AllowComments:     settingsBool(data, "allow_comments", true),
		RequireLogin:      settingsBool(data, "require_login", false),
		AutoApprove:       settingsBool(data, "auto_approve", false),
		AllowAnonymous:    settingsBool(data, "allow_anonymous", true),
		ShowTitle:         settingsBool(data, "show_title", true),
		ShowAuthor:        settingsBool(data, "show_author", true),
		ShowPublishedDate: settingsBool(data, "show_published_date", true),
		ShowAuthorBio:     settingsBool(data, "show_author_bio", true),
		ShowComments:      settingsBool(data, "show_comments", true),
		AuthorProfileID:   settingsUint(data, "author_profile_id"),
	}, nil
}

// AuthorProfileID returns the configured profile schema used for item authors.
func AuthorProfileID(ctx context.Context) (uint, error) {
	settings, err := LoadSettings(ctx)
	if err != nil {
		return 0, err
	}
	return settings.AuthorProfileID, nil
}

func settingsUint(data map[string]any, key string) uint {
	v, ok := data[key]
	if !ok || v == nil {
		return 0
	}
	switch n := v.(type) {
	case float64:
		if n > 0 {
			return uint(n)
		}
	case int:
		if n > 0 {
			return uint(n)
		}
	case uint:
		return n
	case int64:
		if n > 0 {
			return uint(n)
		}
	}
	return 0
}

func settingsBool(data map[string]any, key string, def bool) bool {
	v, ok := data[key]
	if !ok {
		return def
	}
	b, ok := v.(bool)
	if !ok {
		return def
	}
	return b
}
