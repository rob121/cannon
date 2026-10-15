package content

import (
	"context"
	"fmt"
	"strings"

	"github.com/rob121/cannon/internal/models"
	"github.com/rob121/cannon/internal/settings"
)

const SettingsSection = "content"

const (
	DisplayInherit = "inherit"
	DisplayYes     = "yes"
	DisplayNo      = "no"
)

// Settings holds global content configuration.
type Settings struct {
	AllowComments     bool
	RequireLogin      bool
	AutoApprove       bool
	AllowAnonymous    bool
	TitleDisplay       string
	AuthorDisplay      string
	CategoryDisplay    string
	DateDisplay        string
	RelatedDisplay     string
	ShowTitle          bool
	ShowAuthor         bool
	ShowCategoryName   bool
	ShowPublishedDate  bool
	ShowRelatedItems   bool
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
	titleDisplay := settingsDisplayMode(data, "show_title", DisplayYes)
	authorDisplay := settingsDisplayMode(data, "show_author", DisplayYes)
	categoryDisplay := settingsDisplayMode(data, "show_category_name", DisplayYes)
	dateDisplay := settingsDisplayMode(data, "show_published_date", DisplayYes)
	relatedDisplay := settingsDisplayMode(data, "show_related_items", DisplayYes)
	return Settings{
		AllowComments:     settingsBool(data, "allow_comments", true),
		RequireLogin:      settingsBool(data, "require_login", false),
		AutoApprove:       settingsBool(data, "auto_approve", false),
		AllowAnonymous:    settingsBool(data, "allow_anonymous", true),
		TitleDisplay:      titleDisplay,
		AuthorDisplay:     authorDisplay,
		CategoryDisplay:   categoryDisplay,
		DateDisplay:       dateDisplay,
		RelatedDisplay:    relatedDisplay,
		ShowTitle:         ResolveDisplayMode(titleDisplay, true),
		ShowAuthor:        ResolveDisplayMode(authorDisplay, true),
		ShowCategoryName:  ResolveDisplayMode(categoryDisplay, true),
		ShowPublishedDate: ResolveDisplayMode(dateDisplay, true),
		ShowRelatedItems:  ResolveDisplayMode(relatedDisplay, true),
		ShowAuthorBio:     settingsBool(data, "show_author_bio", true),
		ShowComments:      settingsBool(data, "show_comments", true),
		AuthorProfileID:   settingsUint(data, "author_profile_id"),
	}, nil
}

// ItemDisplaySettings controls item title, byline, category, and related-item visibility.
type ItemDisplaySettings struct {
	ShowTitle         bool
	ShowAuthor        bool
	ShowCategoryName  bool
	ShowPublishedDate bool
	ShowRelatedItems  bool
}

// ItemDisplayFromSettings returns the resolved global item display settings.
func ItemDisplayFromSettings(base Settings) ItemDisplaySettings {
	return ItemDisplaySettings{
		ShowTitle:         base.ShowTitle,
		ShowAuthor:        base.ShowAuthor,
		ShowCategoryName:  base.ShowCategoryName,
		ShowPublishedDate: base.ShowPublishedDate,
		ShowRelatedItems:  base.ShowRelatedItems,
	}
}

// AuthorProfileID returns the configured profile schema used for item authors.
func AuthorProfileID(ctx context.Context) (uint, error) {
	settings, err := LoadSettings(ctx)
	if err != nil {
		return 0, err
	}
	return settings.AuthorProfileID, nil
}

// ResolveItemDisplaySettings applies category-scoped display settings to item views.
func ResolveItemDisplaySettings(ctx context.Context, item *models.Item, base Settings) (Settings, error) {
	display := base
	if item == nil || item.Category == nil {
		return display, nil
	}
	itemDisplay, err := ResolveCategoryItemDisplaySettings(ctx, item.Category, base)
	if err != nil {
		return display, err
	}
	display.ShowTitle = itemDisplay.ShowTitle
	display.ShowAuthor = itemDisplay.ShowAuthor
	display.ShowCategoryName = itemDisplay.ShowCategoryName
	display.ShowPublishedDate = itemDisplay.ShowPublishedDate
	display.ShowRelatedItems = itemDisplay.ShowRelatedItems
	return display, nil
}

// NormalizeDisplayMode coerces a display mode into yes, no, or inherit.
func NormalizeDisplayMode(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case DisplayYes, "true", "1", "on":
		return DisplayYes
	case DisplayNo, "false", "0", "off":
		return DisplayNo
	default:
		return DisplayInherit
	}
}

// ResolveDisplayMode applies a tri-state display mode to an inherited value.
func ResolveDisplayMode(mode string, inherited bool) bool {
	switch NormalizeDisplayMode(mode) {
	case DisplayYes:
		return true
	case DisplayNo:
		return false
	default:
		return inherited
	}
}

func settingsDisplayMode(data map[string]any, key, fallback string) string {
	v, ok := data[key]
	if !ok || v == nil {
		return NormalizeDisplayMode(fallback)
	}
	return NormalizeDisplayMode(strings.TrimSpace(toString(v)))
}

func toString(v any) string {
	switch val := v.(type) {
	case string:
		return val
	case bool:
		if val {
			return DisplayYes
		}
		return DisplayNo
	default:
		return strings.TrimSpace(fmt.Sprint(val))
	}
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
