package captcha

import (
	"fmt"
	"strings"

	"github.com/rob121/cannon/extension"
)

const (
	ProviderAny            = extension.ProviderAny
	CaptchaContextLogin    = extension.CaptchaContextLogin
	CaptchaContextRegister = extension.CaptchaContextRegister
	CaptchaContextComment  = extension.CaptchaContextComment
	CaptchaContextForm     = extension.CaptchaContextForm
	TagName                = "captcha"
)

// PlaceholderAttrs configures a captcha placeholder element.
type PlaceholderAttrs struct {
	Context  string
	Provider string
}

// PlaceholderMarkup returns a literal <captcha> element for templates or extension HTML.
// Cannon expands these placeholders after the layout renders and before onAfterRender.
func PlaceholderMarkup(context, provider string) string {
	context = strings.TrimSpace(context)
	if context == "" {
		context = CaptchaContextForm
	}
	provider = strings.TrimSpace(provider)
	if provider == "" {
		provider = ProviderAny
	}
	return fmt.Sprintf(
		`<captcha context=%q provider=%q></captcha>`,
		context,
		provider,
	)
}

// PlaceholderHTML returns escaped-safe placeholder markup for html/template.
func PlaceholderHTML(attrs PlaceholderAttrs) string {
	return PlaceholderMarkup(attrs.Context, attrs.Provider)
}

// NormalizeProvider accepts provider or legacy type attribute values.
func NormalizeProvider(raw string) string {
	raw = strings.TrimSpace(strings.ToLower(raw))
	if raw == "" {
		return ProviderAny
	}
	return raw
}
