package extensions

import (
	"net/http"
	"strings"
)

var wireHeaderAllowlist = map[string]struct{}{
	"accept":            {},
	"accept-language":   {},
	"accept-encoding":   {},
	"content-type":      {},
	"content-length":    {},
	"x-requested-with":  {},
	"cache-control":     {},
	"if-none-match":     {},
	"if-modified-since": {},
	"host":              {},
	"x-forwarded-host":  {},
	"x-forwarded-proto": {},
}

// FilterWireHeaders copies only safe request headers for extension IPC.
func FilterWireHeaders(h http.Header) map[string][]string {
	return filterWireHeaderMap(h)
}

// FilterWireHeaderMap filters a wire header map in place conceptually.
func FilterWireHeaderMap(h map[string][]string) map[string][]string {
	if h == nil {
		return nil
	}
	return filterWireHeaderMap(http.Header(h))
}

func filterWireHeaderMap(h http.Header) map[string][]string {
	if h == nil {
		return nil
	}
	out := make(map[string][]string)
	for key, vals := range h {
		if _, ok := wireHeaderAllowlist[strings.ToLower(key)]; !ok {
			continue
		}
		copied := append([]string(nil), vals...)
		out[key] = copied
	}
	if len(out) == 0 {
		return nil
	}
	return out
}
