package hooks

import (
	"context"
	"net/http"
)

type fireKey struct{}

// FireFunc dispatches a hook to in-process listeners and extensions.
type FireFunc func(ctx context.Context, event string, args map[string]any) (map[string]any, error)

// WithFire attaches a hook dispatcher to context.
func WithFire(ctx context.Context, fn FireFunc) context.Context {
	return context.WithValue(ctx, fireKey{}, fn)
}

// Fire dispatches a hook using the context dispatcher, if present.
func Fire(ctx context.Context, event string, args map[string]any) (map[string]any, error) {
	fn, ok := ctx.Value(fireKey{}).(FireFunc)
	if !ok || fn == nil {
		local, _, err := FireLocal(ctx, nil, event, args)
		return local, err
	}
	return fn(ctx, event, args)
}

// RequestContext builds hook arguments from an HTTP request.
func RequestArgs(r *http.Request) map[string]any {
	if r == nil {
		return map[string]any{}
	}
	return map[string]any{
		"method": r.Method,
		"path":   r.URL.Path,
		"query":  r.URL.RawQuery,
	}
}
