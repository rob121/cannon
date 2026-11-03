package settings

import (
	"github.com/rob121/cannon/internal/content/htmlpolicy"
)

// SanitizeSiteHeadExtra restricts administrator head markup to safe tags.
func SanitizeSiteHeadExtra(raw string) string {
	return htmlpolicy.SanitizeHeadExtra(raw)
}

// SanitizeSEOFormData normalizes SEO settings before persistence.
func SanitizeSEOFormData(data map[string]any) {
	if data == nil {
		return
	}
	if raw, ok := data["site_head_extra"].(string); ok {
		data["site_head_extra"] = SanitizeSiteHeadExtra(raw)
	}
}
