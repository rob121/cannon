package content

import (
	"context"

	"github.com/rob121/cannon/internal/content/htmlpolicy"
	"github.com/rob121/cannon/internal/settings"
)

func htmlPolicyForContext(ctx context.Context) (htmlpolicy.ContentPolicy, []string) {
	if ctx == nil {
		return htmlpolicy.PolicyStandard, append([]string(nil), htmlpolicy.DefaultIframeHosts...)
	}
	raw, err := settings.HTMLContentPolicy(ctx)
	if err != nil {
		return htmlpolicy.PolicyStandard, append([]string(nil), htmlpolicy.DefaultIframeHosts...)
	}
	hosts, err := settings.HTMLIframeAllowHosts(ctx)
	if err != nil || len(hosts) == 0 {
		hosts = append([]string(nil), htmlpolicy.DefaultIframeHosts...)
	}
	return htmlpolicy.NormalizeContentPolicy(raw), hosts
}
