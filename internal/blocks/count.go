package blocks

import (
	"context"
	"strings"

	"github.com/rob121/cannon/internal/extensions"
	"github.com/rob121/cannon/internal/sites"
)

// CountForSpace returns how many admin-assigned blocks are active in a template space
// for the current viewer.
func CountForSpace(ctx context.Context, extMgr *extensions.Manager, space string) (int, error) {
	_ = extMgr
	space = strings.TrimSpace(space)
	if space == "" {
		return 0, nil
	}

	db, err := sites.DB(ctx)
	if err != nil {
		return 0, err
	}

	rows, err := ListForSpace(ctx, db, space)
	if err != nil {
		return 0, err
	}
	return len(rows), nil
}

// PreviewCount adjusts a space count for template-space debug mode (?tp=1): empty
// spaces count as 1 so hasSpace/lenspace still reach the space renderer outline.
func PreviewCount(n int, err error, space string, debugActive bool) (int, error) {
	if debugActive && err == nil && n == 0 && strings.TrimSpace(space) != "" {
		return 1, nil
	}
	return n, err
}
