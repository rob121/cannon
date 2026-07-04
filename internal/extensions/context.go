package extensions

import "context"

type managerKey struct{}

// WithContext stores the site extension manager on the request context.
func WithContext(ctx context.Context, mgr *Manager) context.Context {
	if mgr == nil {
		return ctx
	}
	return context.WithValue(ctx, managerKey{}, mgr)
}

// FromContext returns the extension manager for the current site request.
func FromContext(ctx context.Context) (*Manager, bool) {
	mgr, ok := ctx.Value(managerKey{}).(*Manager)
	return mgr, ok
}
