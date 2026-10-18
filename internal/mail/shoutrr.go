package mail

import (
	"fmt"
	"net"
	"net/url"
	"strconv"
	"strings"
)

// ShoutrrSMTPURL builds a Cannon-compatible smtp:// shoutrrr URL from site mail settings.
func ShoutrrSMTPURL(cfg Settings, to string) (string, error) {
	if !cfg.Configured() {
		return "", fmt.Errorf("mail is not configured (set SMTP host and from email in Configuration → Mail)")
	}
	to = strings.TrimSpace(to)
	if to == "" {
		return "", fmt.Errorf("recipient email is required")
	}
	port := cfg.Port
	if port <= 0 {
		port = 587
	}
	u := &url.URL{
		Scheme: "smtp",
		Host:   net.JoinHostPort(strings.TrimSpace(cfg.Host), strconv.Itoa(port)),
	}
	user := strings.TrimSpace(cfg.Username)
	pass := cfg.Password
	if user != "" {
		u.User = url.UserPassword(user, pass)
	}
	q := url.Values{}
	q.Set("from", strings.TrimSpace(cfg.FromEmail))
	q.Set("to", to)
	u.RawQuery = q.Encode()
	return u.String(), nil
}

// SiteMailRecipient extracts the to= address when url looks like this site's SMTP settings.
func SiteMailRecipient(cfg Settings, rawURL string) (to string, ok bool) {
	rawURL = strings.TrimSpace(rawURL)
	if rawURL == "" || !cfg.Configured() {
		return "", false
	}
	u, err := url.Parse(rawURL)
	if err != nil {
		return "", false
	}
	scheme := strings.ToLower(u.Scheme)
	if scheme != "smtp" && scheme != "smtps" {
		return "", false
	}
	if !strings.EqualFold(u.Hostname(), strings.TrimSpace(cfg.Host)) {
		return "", false
	}
	wantPort := cfg.Port
	if wantPort <= 0 {
		wantPort = 587
	}
	gotPort := u.Port()
	if gotPort == "" {
		gotPort = "587"
	}
	if gotPort != strconv.Itoa(wantPort) {
		return "", false
	}
	from := strings.TrimSpace(u.Query().Get("from"))
	if from != "" && !strings.EqualFold(from, strings.TrimSpace(cfg.FromEmail)) {
		return "", false
	}
	to = strings.TrimSpace(u.Query().Get("to"))
	if to == "" {
		return "", false
	}
	return to, true
}
