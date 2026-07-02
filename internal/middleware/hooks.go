package middleware

import (
	"context"
	"net/http"

	"github.com/rob121/cannon/internal/hooks"
	"github.com/rob121/cannon/internal/sites"
	"github.com/rob121/cannon/internal/user"
)

func (c *Chain) Hooks(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		site, err := sites.FromContext(r.Context())
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		ext := c.Extensions(site)
		userCtx := hookUserContext(r)
		ctx := hooks.WithFire(r.Context(), func(goCtx context.Context, event string, args map[string]any) (map[string]any, error) {
			return ext.FireHook(goCtx, r, userCtx, event, args)
		})
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func hookUserContext(r *http.Request) map[string]any {
	svc, err := user.FromContext(r.Context())
	if err != nil {
		return map[string]any{"authenticated": false}
	}
	ctx, err := svc.Context(r.Context())
	if err != nil {
		return map[string]any{"authenticated": false}
	}
	return ctx
}
