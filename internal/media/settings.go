package media

import (
	"context"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/rob121/cannon/internal/settings"
)

const SettingsSection = "media"

// Settings holds global media manager configuration.
type Settings struct {
	MaxFileSizeBytes int64
	ApprovedExts     []string
	AllowAllTypes    bool
	ImageStyles      []Style
}

// LoadSettings reads media settings from the global settings store.
func LoadSettings(ctx context.Context) (Settings, error) {
	data, err := settings.NewStore().Load(ctx, settings.ScopeGlobal, SettingsSection)
	if err != nil {
		return Settings{}, err
	}
	maxMB, err := settingsInt(data, "max_file_size_mb", 32)
	if err != nil {
		return Settings{}, err
	}
	if maxMB < 1 {
		maxMB = 1
	}
	if maxMB > 512 {
		maxMB = 512
	}
	exts, allowAll := parseApprovedExtensions(settingsString(data, "approved_extensions", defaultApprovedExtensions))
	styles, err := parseImageStyles(data["image_styles"])
	if err != nil {
		styles = DefaultStyles()
	}
	return Settings{
		MaxFileSizeBytes: int64(maxMB) * 1024 * 1024,
		ApprovedExts:     exts,
		AllowAllTypes:    allowAll,
		ImageStyles:      styles,
	}, nil
}

// LoadStyles returns configured image styles, or defaults when unset.
func LoadStyles(ctx context.Context) ([]Style, error) {
	cfg, err := LoadSettings(ctx)
	if err != nil {
		return DefaultStyles(), err
	}
	if len(cfg.ImageStyles) == 0 {
		return DefaultStyles(), nil
	}
	return cfg.ImageStyles, nil
}

const defaultApprovedExtensions = "jpg, jpeg, png, gif, webp, svg, pdf, mp4, webm, mp3, wav, zip"

// ValidateUpload checks filename extension and size against media settings.
func ValidateUpload(cfg Settings, filename string, size int64) error {
	if size < 0 {
		size = 0
	}
	if cfg.MaxFileSizeBytes > 0 && size > cfg.MaxFileSizeBytes {
		return fmt.Errorf("file exceeds the maximum upload size of %d MB", cfg.MaxFileSizeBytes/(1024*1024))
	}
	if cfg.AllowAllTypes || len(cfg.ApprovedExts) == 0 {
		return nil
	}
	ext := strings.ToLower(strings.TrimPrefix(filepath.Ext(filename), "."))
	if ext == "" {
		return fmt.Errorf("files must have an approved extension: %s", strings.Join(cfg.ApprovedExts, ", "))
	}
	for _, allowed := range cfg.ApprovedExts {
		if ext == allowed {
			return nil
		}
	}
	return fmt.Errorf("file type %q is not allowed; approved extensions: %s", ext, strings.Join(cfg.ApprovedExts, ", "))
}

// ApprovedExtensionsLabel returns a human-readable list for UI hints.
func (cfg Settings) ApprovedExtensionsLabel() string {
	if cfg.AllowAllTypes || len(cfg.ApprovedExts) == 0 {
		return "all file types"
	}
	return strings.Join(cfg.ApprovedExts, ", ")
}

func parseApprovedExtensions(raw string) ([]string, bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return parseApprovedExtensions(defaultApprovedExtensions)
	}
	if raw == "*" {
		return nil, true
	}
	parts := strings.FieldsFunc(raw, func(r rune) bool {
		return r == ',' || r == ';' || r == ' ' || r == '\n' || r == '\t'
	})
	out := make([]string, 0, len(parts))
	seen := map[string]struct{}{}
	for _, part := range parts {
		part = strings.ToLower(strings.TrimSpace(part))
		part = strings.TrimPrefix(part, ".")
		if part == "" || part == "*" {
			if part == "*" {
				return nil, true
			}
			continue
		}
		if _, ok := seen[part]; ok {
			continue
		}
		seen[part] = struct{}{}
		out = append(out, part)
	}
	return out, false
}

func settingsString(data map[string]any, key, def string) string {
	v, ok := data[key]
	if !ok || v == nil {
		return def
	}
	if s, ok := v.(string); ok {
		if strings.TrimSpace(s) == "" {
			return def
		}
		return strings.TrimSpace(s)
	}
	return strings.TrimSpace(fmt.Sprint(v))
}

func settingsInt(data map[string]any, key string, def int) (int, error) {
	v, ok := data[key]
	if !ok || v == nil {
		return def, nil
	}
	return anyToInt(v, def), nil
}

func parseImageStyles(raw any) ([]Style, error) {
	if raw == nil {
		return DefaultStyles(), nil
	}
	var rows []any
	switch typed := raw.(type) {
	case []any:
		rows = typed
	case []Style:
		return ValidateStyles(typed)
	case []map[string]any:
		rows = make([]any, len(typed))
		for i, m := range typed {
			rows[i] = m
		}
	default:
		return DefaultStyles(), nil
	}
	if len(rows) == 0 {
		return DefaultStyles(), nil
	}
	out := make([]Style, 0, len(rows))
	for _, row := range rows {
		m, ok := row.(map[string]any)
		if !ok {
			continue
		}
		out = append(out, Style{
			Key:    settingsString(m, "key", ""),
			Width:  anyToInt(m["width"], 0),
			Height: anyToInt(m["height"], 0),
			Mode:   settingsString(m, "mode", ModeFit),
		})
	}
	return ValidateStyles(out)
}

func anyToInt(v any, def int) int {
	switch n := v.(type) {
	case float64:
		return int(n)
	case float32:
		return int(n)
	case int:
		return n
	case int64:
		return int(n)
	case string:
		if i, err := strconv.Atoi(strings.TrimSpace(n)); err == nil {
			return i
		}
		return def
	default:
		return def
	}
}
