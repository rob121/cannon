package extension

import (
	"net/url"
	"strings"
)

// SettingSitePublicBase is the configured public site origin (scheme + host, no trailing slash).
const SettingSitePublicBase = "site_public_base"

// PublicRequestURL builds an absolute public URL for path using the wire request and site settings.
func PublicRequestURL(req WireRequest, path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return ""
	}
	if strings.HasPrefix(path, "http://") || strings.HasPrefix(path, "https://") {
		return path
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	if u, err := url.Parse(req.URL); err == nil && u.Scheme != "" && u.Host != "" {
		return u.Scheme + "://" + u.Host + path
	}
	if origin := requestOrigin(req); origin != "" {
		return strings.TrimRight(origin, "/") + path
	}
	if base := normalizeSitePublicBase(Setting(req, SettingSitePublicBase)); base != "" {
		return base + path
	}
	return path
}

func requestOrigin(req WireRequest) string {
	host := wireHeaderFirst(req, "X-Forwarded-Host")
	if host == "" {
		host = wireHeaderFirst(req, "Host")
	}
	host = strings.TrimSpace(host)
	if host == "" {
		return ""
	}
	if i := strings.Index(host, ","); i >= 0 {
		host = strings.TrimSpace(host[:i])
	}
	scheme := strings.ToLower(wireHeaderFirst(req, "X-Forwarded-Proto"))
	if scheme != "http" && scheme != "https" {
		scheme = "http"
	}
	return scheme + "://" + host
}

func wireHeaderFirst(req WireRequest, name string) string {
	if req.Header == nil {
		return ""
	}
	for key, values := range req.Header {
		if !strings.EqualFold(key, name) || len(values) == 0 {
			continue
		}
		return strings.TrimSpace(values[0])
	}
	return ""
}

func normalizeSitePublicBase(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	if !strings.Contains(raw, "://") {
		raw = "http://" + raw
	}
	return strings.TrimRight(raw, "/")
}
