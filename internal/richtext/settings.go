package richtext

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/rob121/cannon/internal/settings"
)

const SettingsSection = "richtext"

const (
	EnterKeyParagraph  = "paragraph"
	EnterKeyLineBreak  = "line_break"
	SpacingDefault     = "default"
	SpacingCompact     = "compact"
)

// ClientConfig is passed to cannon-richtext.js in the page.
type ClientConfig struct {
	EnterKeyMode     string   `json:"enterKeyMode"`
	ParagraphSpacing string   `json:"paragraphSpacing"`
	IntroToolbar     []string `json:"introToolbar"`
	BodyToolbar      []string `json:"bodyToolbar"`
}

// Settings holds stored rich text editor preferences.
type Settings struct {
	EnterKeyMode     string
	ParagraphSpacing string
	IntroToolbar     []string
	BodyToolbar      []string
}

var defaultIntroToolbar = []string{
	"bold", "italic", "link", "|",
	"bulletedList", "numberedList", "|",
	"cannonMediaImage", "|",
	"sourceEditing", "|", "undo", "redo",
}

var defaultBodyToolbar = []string{
	"heading", "|",
	"bold", "italic", "link", "|",
	"bulletedList", "numberedList", "blockQuote", "|",
	"cannonMediaImage", "insertTable", "mediaEmbed", "|",
	"sourceEditing", "|", "undo", "redo",
}

var toolbarItemAllow = map[string]bool{
	"heading": true, "bold": true, "italic": true, "link": true,
	"bulletedList": true, "numberedList": true, "blockQuote": true,
	"uploadImage": true, "insertImage": true, "cannonMediaImage": true,
	"insertTable": true, "mediaEmbed": true,
	"sourceEditing": true, "undo": true, "redo": true, "|": true,
}

// LoadSettings reads rich text settings from the global settings store.
func LoadSettings(ctx context.Context) (Settings, error) {
	data, err := settings.NewStore().Load(ctx, settings.ScopeGlobal, SettingsSection)
	if err != nil {
		return Settings{}, err
	}
	enterMode := strings.TrimSpace(stringSetting(data, "enter_key_mode", EnterKeyParagraph))
	if enterMode != EnterKeyLineBreak {
		enterMode = EnterKeyParagraph
	}
	spacing := strings.TrimSpace(stringSetting(data, "paragraph_spacing", SpacingDefault))
	if spacing != SpacingCompact {
		spacing = SpacingDefault
	}
	return Settings{
		EnterKeyMode:     enterMode,
		ParagraphSpacing: spacing,
		IntroToolbar:     parseToolbar(stringSetting(data, "toolbar_intro", ""), toolbarItemAllow, defaultIntroToolbar),
		BodyToolbar:      parseToolbar(stringSetting(data, "toolbar_body", ""), toolbarItemAllow, defaultBodyToolbar),
	}, nil
}

// ClientConfigFromContext builds the JSON payload for cannon-richtext.js.
func ClientConfigFromContext(ctx context.Context) (ClientConfig, error) {
	cfg, err := LoadSettings(ctx)
	if err != nil {
		return ClientConfig{}, err
	}
	return ClientConfig{
		EnterKeyMode:     cfg.EnterKeyMode,
		ParagraphSpacing: cfg.ParagraphSpacing,
		IntroToolbar:     cfg.IntroToolbar,
		BodyToolbar:      cfg.BodyToolbar,
	}, nil
}

// ClientConfigJSON returns marshaled client config or empty object on error.
func ClientConfigJSON(ctx context.Context) string {
	cfg, err := ClientConfigFromContext(ctx)
	if err != nil {
		raw, _ := json.Marshal(ClientConfig{
			EnterKeyMode:     EnterKeyParagraph,
			ParagraphSpacing: SpacingDefault,
			IntroToolbar:     append([]string(nil), defaultIntroToolbar...),
			BodyToolbar:      append([]string(nil), defaultBodyToolbar...),
		})
		return string(raw)
	}
	raw, err := json.Marshal(cfg)
	if err != nil {
		return "{}"
	}
	return string(raw)
}

func stringSetting(data map[string]any, key, fallback string) string {
	if data == nil {
		return fallback
	}
	v, ok := data[key]
	if !ok || v == nil {
		return fallback
	}
	switch t := v.(type) {
	case string:
		if strings.TrimSpace(t) == "" {
			return fallback
		}
		return strings.TrimSpace(t)
	default:
		s := strings.TrimSpace(strings.TrimSpace(fmt.Sprint(t)))
		if s == "" {
			return fallback
		}
		return s
	}
}

func parseToolbar(raw string, allow map[string]bool, fallback []string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return append([]string(nil), fallback...)
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		item := strings.TrimSpace(part)
		if item == "" {
			continue
		}
		if !allow[item] {
			continue
		}
		out = append(out, item)
	}
	if len(out) == 0 {
		return append([]string(nil), fallback...)
	}
	return out
}
