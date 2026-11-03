package session

import (
	"net/http"
	"strings"

	"github.com/rob121/cannon/internal/config"
)

// CookieSecure reports whether the session cookie should include the Secure flag.
func CookieSecure(r *http.Request, cfg *config.SessionConfig) bool {
	if cfg != nil && cfg.CookieSecure {
		return true
	}
	if r == nil {
		return false
	}
	if r.TLS != nil {
		return true
	}
	if proto := strings.TrimSpace(r.Header.Get("X-Forwarded-Proto")); strings.EqualFold(proto, "https") {
		return true
	}
	return false
}

// CookieSecureForRequest uses this store's session config.
func (s *Store) CookieSecureForRequest(r *http.Request) bool {
	return CookieSecure(r, s.cfg)
}
