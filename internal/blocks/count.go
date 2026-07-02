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
