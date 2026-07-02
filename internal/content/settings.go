package content

import (
	"context"

	"github.com/rob121/cannon/internal/settings"
)

const SettingsSection = "content"

// Settings holds global content configuration.
type Settings struct {
	AllowComments  bool
	RequireLogin   bool
	AutoApprove    bool
	AllowAnonymous bool
}

// LoadSettings reads content settings from the global settings store.
func LoadSettings(ctx context.Context) (Settings, error) {
	store := settings.NewStore()
	data, err := store.Load(ctx, settings.ScopeGlobal, SettingsSection)
	if err != nil {
		return Settings{}, err
	}
	return Settings{
		AllowComments:  settingsBool(data, "allow_comments", true),
		RequireLogin:   settingsBool(data, "require_login", false),
		AutoApprove:    settingsBool(data, "auto_approve", false),
		AllowAnonymous: settingsBool(data, "allow_anonymous", true),
	}, nil
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
