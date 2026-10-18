package cache

import (
	"context"

	"github.com/rob121/cannon/internal/hooks"
)

func init() {
	registerPublicInvalidationHooks()
}

func registerPublicInvalidationHooks() {
	flush := func(ctx context.Context, _ *hooks.Event) (*hooks.Result, error) {
		InvalidateSitePublic(SiteIDFromContext(ctx))
		return nil, nil
	}
	for _, event := range []string{
		hooks.OnItemAfterSave,
		hooks.OnItemTrash,
		hooks.OnItemRestore,
		hooks.OnItemAfterDelete,
		hooks.OnCategoryAfterSave,
		hooks.OnCommentAfterSave,
		hooks.OnRevisionRestore,
		hooks.OnSettingsSave,
		hooks.OnMediaUpload,
		hooks.OnMediaDelete,
	} {
		hooks.Register(event, flush)
	}
}
