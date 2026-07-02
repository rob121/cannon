package themes

import (
	"context"

	"github.com/rob121/cannon/internal/settings"
)

// SelectionFromContext loads the active frontend and admin themes for the current site.
func SelectionFromContext(ctx context.Context) (Selection, error) {
	sel := Selection{}
	frontend, err := settings.FrontendTheme(ctx)
	if err != nil {
		return sel, err
	}
	admin, err := settings.AdminTheme(ctx)
	if err != nil {
		return sel, err
	}
	sel.Frontend = frontend
	sel.Admin = admin
	return sel.Normalize(), nil
}
