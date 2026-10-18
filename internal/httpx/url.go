package httpx

import (
	"net/http"
	"strings"
)

// Scheme returns http or https for the request (honors X-Forwarded-Proto).
func Scheme(r *http.Request) string {
	if r != nil && (r.TLS != nil || strings.EqualFold(r.Header.Get("X-Forwarded-Proto"), "https")) {
		return "https"
	}
	return "http"
}

// Origin returns scheme://host for the request (honors X-Forwarded-Proto).
func Origin(r *http.Request) string {
	if r == nil || strings.TrimSpace(r.Host) == "" {
		return ""
	}
	return Scheme(r) + "://" + r.Host
}

// AbsoluteURL builds an absolute URL for the current request host.
func AbsoluteURL(r *http.Request, path string) string {
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
	if r == nil || strings.TrimSpace(r.Host) == "" {
		return path
	}
	return Origin(r) + path
}
