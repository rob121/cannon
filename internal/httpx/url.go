package httpx

import (
	"net/http"
	"strings"
)

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
	scheme := "http"
	if r.TLS != nil || strings.EqualFold(r.Header.Get("X-Forwarded-Proto"), "https") {
		scheme = "https"
	}
	return scheme + "://" + r.Host + path
}
