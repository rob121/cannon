package blocks

import (
	"context"
	"fmt"
	"html"
	"net/http"
	"strings"

	"github.com/rob121/cannon/internal/settings"
)

const debugSpacesQueryParam = "tp"

// DebugSpacesActive reports whether template space debug outlines should render.
// Requires the global debug_template_spaces setting and tp=1 in the request URL.
func DebugSpacesActive(ctx context.Context, r *http.Request) bool {
	if r == nil || r.URL.Query().Get(debugSpacesQueryParam) != "1" {
		return false
	}
	enabled, err := settings.GlobalBool(ctx, settings.SectionGeneral, "debug_template_spaces")
	if err != nil {
		return false
	}
	return enabled
}

// WrapDebugSpace wraps rendered space HTML in a visible debug outline.
func WrapDebugSpace(name, content string) string {
	label := strings.TrimSpace(name)
	if label == "" {
		label = "?"
	}
	return fmt.Sprintf(
		`<div class="cannon-space-debug" style="border:1px dashed #FF3300"><div class="cannon-space-debug-label" style="color:#FF3300;font-size:12px;font-family:monospace;padding:2px 4px">%s</div>%s</div>`,
		html.EscapeString(label),
		content,
	)
}
