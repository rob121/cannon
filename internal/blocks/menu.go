package blocks

import (
	"context"
	"strings"

	"github.com/rob121/cannon/internal/router"
)

const menuBlockMaxDepth = 3

// RenderMenuVerticalBlock renders a stacked Bootstrap nav menu block.
func RenderMenuVerticalBlock(ctx context.Context, meta Metadata, render FragmentRenderer) (string, error) {
	return renderMenuBlock(ctx, meta, MenuVerticalBlockTemplate, render)
}

// RenderMenuHorizontalBlock renders a horizontal Bootstrap nav menu block.
func RenderMenuHorizontalBlock(ctx context.Context, meta Metadata, render FragmentRenderer) (string, error) {
	return renderMenuBlock(ctx, meta, MenuHorizontalBlockTemplate, render)
}

func renderMenuBlock(ctx context.Context, meta Metadata, tmpl string, render FragmentRenderer) (string, error) {
	menuName := strings.TrimSpace(meta.MenuName)
	if menuName == "" {
		return "", nil
	}
	items, err := router.MenuDataWithDepth(ctx, menuName, menuBlockMaxDepth)
	if err != nil {
		return "", err
	}
	if len(items) == 0 {
		return "", nil
	}
	if render == nil {
		return "", nil
	}
	html, err := render(tmpl, map[string]any{
		"Items": items,
		"Class": strings.TrimSpace(meta.MenuClass),
	})
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(html), nil
}
