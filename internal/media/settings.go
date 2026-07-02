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
	return Settings{
		MaxFileSizeBytes: int64(maxMB) * 1024 * 1024,
		ApprovedExts:     exts,
		AllowAllTypes:    allowAll,
	}, nil
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
	switch n := v.(type) {
	case float64:
		return int(n), nil
	case int:
		return n, nil
	case int64:
		return int(n), nil
	case string:
		if i, err := strconv.Atoi(strings.TrimSpace(n)); err == nil {
			return i, nil
		}
		return def, nil
	default:
		return def, nil
	}
}
