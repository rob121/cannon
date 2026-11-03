package htmlpolicy

import (
	"regexp"
	"strings"
	"sync"

	"github.com/microcosm-cc/bluemonday"
)

// ContentPolicy selects how rich HTML from the admin editor is sanitized.
type ContentPolicy string

const (
	PolicyStrict   ContentPolicy = "strict"
	PolicyStandard ContentPolicy = "standard"
	PolicyTrusted  ContentPolicy = "trusted"
)

// DefaultIframeHosts are used when the site has not configured an allowlist.
var DefaultIframeHosts = []string{
	"www.youtube.com",
	"youtube.com",
	"www.youtube-nocookie.com",
	"youtube-nocookie.com",
	"player.vimeo.com",
	"vimeo.com",
}

var (
	policyMu sync.RWMutex
	policyCache = map[string]*bluemonday.Policy{}
)

// NormalizeContentPolicy maps stored settings to a known policy (default standard).
func NormalizeContentPolicy(raw string) ContentPolicy {
	switch ContentPolicy(strings.ToLower(strings.TrimSpace(raw))) {
	case PolicyStrict:
		return PolicyStrict
	case PolicyTrusted:
		return PolicyTrusted
	default:
		return PolicyStandard
	}
}

// SanitizeContentHTML applies the configured content policy to an HTML fragment.
func SanitizeContentHTML(policy ContentPolicy, iframeHosts []string, html string) string {
	html = strings.TrimSpace(html)
	if html == "" {
		return ""
	}
	p := contentPolicy(policy, iframeHosts)
	return strings.TrimSpace(p.Sanitize(html))
}

// SanitizeHeadExtra restricts additional site head markup to safe meta/link tags.
func SanitizeHeadExtra(html string) string {
	html = strings.TrimSpace(html)
	if html == "" {
		return ""
	}
	return strings.TrimSpace(headExtraPolicy().Sanitize(html))
}

func contentPolicy(policy ContentPolicy, iframeHosts []string) *bluemonday.Policy {
	hosts := normalizeHosts(iframeHosts)
	if len(hosts) == 0 {
		hosts = append([]string(nil), DefaultIframeHosts...)
	}
	key := string(policy) + "\x00" + strings.Join(hosts, ",")
	policyMu.RLock()
	if p, ok := policyCache[key]; ok {
		policyMu.RUnlock()
		return p
	}
	policyMu.RUnlock()

	policyMu.Lock()
	defer policyMu.Unlock()
	if p, ok := policyCache[key]; ok {
		return p
	}
	var p *bluemonday.Policy
	switch policy {
	case PolicyStrict:
		p = strictContentPolicy()
	case PolicyTrusted:
		p = trustedContentPolicy(hosts)
	default:
		p = standardContentPolicy(hosts)
	}
	policyCache[key] = p
	return p
}

func strictContentPolicy() *bluemonday.Policy {
	p := bluemonday.NewPolicy()
	p.AllowElements(
		"p", "br", "strong", "b", "em", "i", "u", "s", "del",
		"a", "ul", "ol", "li", "blockquote", "h2", "h3", "h4",
	)
	p.AllowAttrs("href").OnElements("a")
	p.AllowAttrs("title").OnElements("a")
	p.AllowStandardURLs()
	return p
}

func standardContentPolicy(iframeHosts []string) *bluemonday.Policy {
	p := bluemonday.UGCPolicy()
	addIframeEmbedPolicy(p, iframeHosts)
	return p
}

func trustedContentPolicy(iframeHosts []string) *bluemonday.Policy {
	p := bluemonday.UGCPolicy()
	p.AllowAttrs("class").Matching(regexp.MustCompile(`^[\w\s\-:,]+$`)).Globally()
	addIframeEmbedPolicy(p, iframeHosts)
	return p
}

func addIframeEmbedPolicy(p *bluemonday.Policy, iframeHosts []string) {
	p.AllowElements("iframe")
	p.AllowAttrs("width", "height", "frameborder").Matching(bluemonday.Integer).OnElements("iframe")
	p.AllowAttrs("allow").Matching(regexp.MustCompile(`^(([\p{L}\p{N}_-]+)(; )?)+$`)).OnElements("iframe")
	p.AllowAttrs("allowfullscreen").OnElements("iframe")
	p.AllowAttrs("title").OnElements("iframe")
	p.AllowAttrs("loading").Matching(regexp.MustCompile(`(?i)^(lazy|eager)$`)).OnElements("iframe")
	p.AllowAttrs("referrerpolicy").OnElements("iframe")
	p.AllowAttrs("src").Matching(iframeSrcRegexp(iframeHosts)).OnElements("iframe")
}

func iframeSrcRegexp(hosts []string) *regexp.Regexp {
	allowed := normalizeHosts(hosts)
	if len(allowed) == 0 {
		return regexp.MustCompile(`^$`)
	}
	parts := make([]string, 0, len(allowed))
	for _, h := range allowed {
		escaped := regexp.QuoteMeta(h)
		parts = append(parts, `(?i)https://([a-z0-9.-]*\.)?`+escaped+`(/|$|\?)`)
	}
	return regexp.MustCompile(`^(` + strings.Join(parts, "|") + `).*$`)
}

func normalizeHosts(hosts []string) []string {
	out := make([]string, 0, len(hosts))
	seen := make(map[string]struct{}, len(hosts))
	for _, h := range hosts {
		h = strings.ToLower(strings.TrimSpace(h))
		h = strings.TrimPrefix(h, "https://")
		h = strings.TrimPrefix(h, "http://")
		if slash := strings.IndexByte(h, '/'); slash >= 0 {
			h = h[:slash]
		}
		if h == "" {
			continue
		}
		if _, ok := seen[h]; ok {
			continue
		}
		seen[h] = struct{}{}
		out = append(out, h)
	}
	return out
}

func headExtraPolicy() *bluemonday.Policy {
	p := bluemonday.NewPolicy()
	p.AllowElements("meta", "link")
	p.AllowAttrs(
		"name", "content", "property", "http-equiv", "charset",
		"rel", "href", "type", "sizes", "media", "crossorigin", "referrerpolicy",
	).Globally()
	p.AllowURLSchemes("https", "http")
	return p
}
