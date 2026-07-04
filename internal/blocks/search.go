package blocks

import (
	"context"
	"strings"

	"github.com/rob121/cannon/internal/content"
)

// RenderSearchHorizontalBlock renders an inline keyword search form.
func RenderSearchHorizontalBlock(ctx context.Context, row BlockRow, meta Metadata, render FragmentRenderer) (string, error) {
	return renderSearchBlock(ctx, row, meta, SearchHorizontalBlockTemplate, render)
}

// RenderSearchVerticalBlock renders a stacked keyword search form.
func RenderSearchVerticalBlock(ctx context.Context, row BlockRow, meta Metadata, render FragmentRenderer) (string, error) {
	return renderSearchBlock(ctx, row, meta, SearchVerticalBlockTemplate, render)
}

func renderSearchBlock(ctx context.Context, row BlockRow, meta Metadata, tmpl string, render FragmentRenderer) (string, error) {
	_ = ctx
	if render == nil {
		return "", nil
	}
	html, err := render(tmpl, searchBlockData(ctx, row.BlockID, meta))
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(html), nil
}

func searchBlockData(ctx context.Context, blockID uint, meta Metadata) map[string]any {
	placeholder := strings.TrimSpace(meta.SearchPlaceholder)
	if placeholder == "" {
		placeholder = "Search…"
	}
	button := strings.TrimSpace(meta.SearchButton)
	if button == "" {
		button = "Search"
	}
	return map[string]any{
		"Action":      content.SearchURLForContext(ctx, ""),
		"Placeholder": placeholder,
		"Button":      button,
		"Label":       strings.TrimSpace(meta.SearchLabel),
		"Class":       strings.TrimSpace(meta.SearchClass),
		"BlockID":     blockID,
	}
}
