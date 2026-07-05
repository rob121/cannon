package realtime

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/rob121/cannon/internal/hooks"
	"github.com/rob121/cannon/internal/httpreq"
	"github.com/rob121/cannon/internal/settings"
	"github.com/rob121/cannon/internal/sites"
	"github.com/rob121/cannon/internal/user"
)

func init() {
	hooks.Register(hooks.OnPrepareDocumentBody, analyticsDocumentBodyHook)
}

func analyticsDocumentBodyHook(ctx context.Context, e *hooks.Event) (*hooks.Result, error) {
	if e == nil {
		return nil, nil
	}
	if surface, _ := e.Arguments["context"].(string); surface == "admin" {
		return nil, nil
	}
	fragment, ok := BodyFragment(ctx, e.Request)
	if !ok || strings.TrimSpace(fragment) == "" {
		return nil, nil
	}
	return &hooks.Result{Arguments: map[string]any{"body_html": fragment}}, nil
}

// BodyFragment returns analytics script markup for enabled public pages.
func BodyFragment(ctx context.Context, r *http.Request) (string, bool) {
	if ActiveHub() == nil {
		return "", false
	}
	enabled, err := settings.AnalyticsEnabled(ctx)
	if err != nil || !enabled {
		return "", false
	}
	authOnly, err := settings.AnalyticsAuthenticatedOnly(ctx)
	if err == nil && authOnly {
		svc, err := user.FromContext(ctx)
		if err != nil {
			return "", false
		}
		if _, ok := svc.CurrentID(); !ok {
			return "", false
		}
	}
	site, err := sites.FromContext(ctx)
	if err != nil {
		return "", false
	}
	scheme, host := endpointHost(ctx, r, site.Host)
	raw, err := ConfigJSON(site.ID, scheme, host, false)
	if err != nil {
		return "", false
	}
	return fmt.Sprintf(
		`<script src="https://cdn.jsdelivr.net/npm/centrifuge@5.2.2/dist/centrifuge.js"></script>`+"\n"+
			`<script type="application/json" id="cannon-realtime-config">%s</script>`+"\n"+
			`<script src="/theme/site-analytics.js?v=1" defer></script>`,
		escapeScriptJSON(string(raw)),
	), true
}

func escapeScriptJSON(s string) string {
	return strings.ReplaceAll(s, "</", `\u003c/`)
}

func endpointHost(ctx context.Context, r *http.Request, siteHost string) (scheme, host string) {
	if r == nil {
		if req, ok := httpreq.FromContext(ctx); ok {
			r = req
		}
	}
	if r != nil && strings.TrimSpace(r.Host) != "" {
		return requestScheme(r), r.Host
	}
	host = strings.TrimSpace(siteHost)
	if host == "" {
		host = "localhost"
	}
	return "http", host
}

func requestScheme(r *http.Request) string {
	if r.TLS != nil || strings.EqualFold(r.Header.Get("X-Forwarded-Proto"), "https") {
		return "https"
	}
	return "http"
}
