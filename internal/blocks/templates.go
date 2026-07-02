package blocks

import "strings"

const (
	// DefaultWrapperTemplate is the standard block output wrapper (always uses a card).
	DefaultWrapperTemplate = "default/partials/blocks/default.html"
	// CardWrapperTemplate is the default wrapper when none is selected (card when show name is on).
	CardWrapperTemplate = "default/partials/blocks/card.html"
	// PlainWrapperTemplate renders block body without extra markup.
	PlainWrapperTemplate = "default/partials/blocks/plain.html"
	// LoginBlockTemplate renders login-type blocks.
	LoginBlockTemplate = "default/partials/blocks/login.html"
	// MenuVerticalBlockTemplate renders vertical menu blocks.
	MenuVerticalBlockTemplate = "default/partials/blocks/menu-vertical.html"
	// MenuHorizontalBlockTemplate renders horizontal menu blocks.
	MenuHorizontalBlockTemplate = "default/partials/blocks/menu-horizontal.html"
)

var legacyBlockTemplates = map[string]string{
	"default/partials/block.html":       DefaultWrapperTemplate,
	"default/partials/login-block.html": LoginBlockTemplate,
}

// ResolveBlockTemplate maps legacy block template paths to the current location.
func ResolveBlockTemplate(name string) string {
	name = strings.TrimSpace(name)
	if mapped, ok := legacyBlockTemplates[name]; ok {
		return mapped
	}
	return name
}
