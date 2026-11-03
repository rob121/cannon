package fetchurl

import (
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// ErrBlockedURL is returned when a URL targets a disallowed host or network.
var ErrBlockedURL = errors.New("URL is not allowed")

// Options configures outbound HTTP(S) fetch validation.
type Options struct {
	AllowHTTP       bool
	AllowPrivateIPs bool // test/local only; never set in production code paths
}

// ValidateHTTPURL ensures raw is a fetchable http(s) URL that does not target private or local networks.
func ValidateHTTPURL(raw string, opts Options) (*url.URL, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, fmt.Errorf("URL is required")
	}
	u, err := url.Parse(raw)
	if err != nil {
		return nil, fmt.Errorf("invalid URL")
	}
	switch strings.ToLower(u.Scheme) {
	case "https":
	case "http":
		if !opts.AllowHTTP {
			return nil, fmt.Errorf("URL must use https")
		}
	default:
		return nil, fmt.Errorf("URL must use http or https")
	}
	if strings.TrimSpace(u.Host) == "" {
		return nil, fmt.Errorf("invalid URL host")
	}
	host := strings.ToLower(u.Hostname())
	if err := validateResolvedHost(host, opts.AllowPrivateIPs); err != nil {
		return nil, err
	}
	return u, nil
}

// ValidateGitRemoteURL validates git clone/list remote URLs (https or git@).
func ValidateGitRemoteURL(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", fmt.Errorf("repository URL is required")
	}
	if strings.HasPrefix(raw, "git@") {
		host := gitSSHHost(raw)
		if host == "" {
			return "", fmt.Errorf("invalid git SSH URL")
		}
		if err := validateResolvedHost(host, false); err != nil {
			return "", err
		}
		return raw, nil
	}
	u, err := ValidateHTTPURL(raw, Options{AllowHTTP: true})
	if err != nil {
		return "", err
	}
	return u.String(), nil
}

func gitSSHHost(raw string) string {
	rest := strings.TrimPrefix(raw, "git@")
	colon := strings.Index(rest, ":")
	if colon <= 0 {
		return ""
	}
	return strings.ToLower(strings.TrimSpace(rest[:colon]))
}

func validateResolvedHost(host string, allowPrivateIPs bool) error {
	if host == "" {
		return ErrBlockedURL
	}
	if isBlockedHostname(host) {
		return ErrBlockedURL
	}
	if ip := net.ParseIP(host); ip != nil {
		if !allowPrivateIPs && isNonPublicIP(ip) {
			return ErrBlockedURL
		}
		return nil
	}
	ips, err := net.LookupIP(host)
	if err != nil {
		return fmt.Errorf("resolve host: %w", err)
	}
	if len(ips) == 0 {
		return ErrBlockedURL
	}
	for _, ip := range ips {
		if !allowPrivateIPs && isNonPublicIP(ip) {
			return ErrBlockedURL
		}
	}
	return nil
}

func isBlockedHostname(host string) bool {
	host = strings.ToLower(strings.TrimSpace(host))
	host = strings.TrimSuffix(host, ".")
	switch host {
	case "localhost", "localhost.localdomain", "metadata", "metadata.google.internal":
		return true
	}
	if strings.HasSuffix(host, ".local") || strings.HasSuffix(host, ".internal") {
		return true
	}
	return false
}

func isNonPublicIP(ip net.IP) bool {
	if ip == nil {
		return true
	}
	if ip.IsLoopback() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() || ip.IsPrivate() || ip.IsUnspecified() {
		return true
	}
	if ip4 := ip.To4(); ip4 != nil {
		// CGNAT and metadata-adjacent ranges.
		if ip4[0] == 169 && ip4[1] == 254 {
			return true
		}
		if ip4[0] == 0 {
			return true
		}
	}
	return false
}

// SafeHTTPClient returns a client that validates every redirect target.
func SafeHTTPClient(timeout time.Duration, opts Options) *http.Client {
	if timeout <= 0 {
		timeout = 2 * time.Minute
	}
	return &http.Client{
		Timeout: timeout,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 10 {
				return fmt.Errorf("too many redirects")
			}
			if _, err := ValidateHTTPURL(req.URL.String(), opts); err != nil {
				return ErrBlockedURL
			}
			return nil
		},
	}
}
