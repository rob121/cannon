package httpreq

import (
	"context"
	"net/http"
)

type requestKey struct{}

// WithContext stores the current HTTP request on a context.
func WithContext(ctx context.Context, r *http.Request) context.Context {
	if r == nil {
		return ctx
	}
	return context.WithValue(ctx, requestKey{}, r)
}

// FromContext returns the HTTP request attached to the context.
func FromContext(ctx context.Context) (*http.Request, bool) {
	r, ok := ctx.Value(requestKey{}).(*http.Request)
	return r, ok
}
