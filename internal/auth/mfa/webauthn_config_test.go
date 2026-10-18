package mfa

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/rob121/cannon/internal/config"
	"github.com/rob121/cannon/internal/sites"
)

func TestSchemeTwinOrigin(t *testing.T) {
	if got := schemeTwinOrigin("http://example.com"); got != "https://example.com" {
		t.Fatalf("schemeTwinOrigin(http) = %q", got)
	}
	if got := schemeTwinOrigin("https://example.com:8443"); got != "http://example.com:8443" {
		t.Fatalf("schemeTwinOrigin(https) = %q", got)
	}
}

func TestWebAuthnInstanceIncludesRequestOrigin(t *testing.T) {
	ctx := sites.WithContext(context.Background(), &config.SiteConfig{
		ID:   "demo",
		Name: "Demo",
		Host: "http://example.com",
	})
	req := httptest.NewRequest(http.MethodPost, "https://example.com/account/security/passkey/begin", nil)
	req.Host = "example.com"
	req.Header.Set("X-Forwarded-Proto", "https")

	wa, err := WebAuthnInstance(ctx, req)
	if err != nil {
		t.Fatalf("WebAuthnInstance: %v", err)
	}
	if wa.Config.RPID != "example.com" {
		t.Fatalf("RPID = %q", wa.Config.RPID)
	}
	origins := map[string]bool{}
	for _, o := range wa.Config.RPOrigins {
		origins[o] = true
	}
	if !origins["https://example.com"] {
		t.Fatalf("RPOrigins missing https request origin: %v", wa.Config.RPOrigins)
	}
	if !origins["http://example.com"] {
		t.Fatalf("RPOrigins missing configured http origin: %v", wa.Config.RPOrigins)
	}
}

func TestRequestHostnameStripsPort(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "http://localhost:8001/", nil)
	req.Host = "localhost:8001"
	if got := requestHostname(req); got != "localhost" {
		t.Fatalf("requestHostname = %q", got)
	}
}

func TestWebAuthnInstanceRejectsIPAddressRPID(t *testing.T) {
	ctx := sites.WithContext(context.Background(), &config.SiteConfig{
		ID:   "demo",
		Name: "Demo",
		Host: "http://127.0.0.1:8001",
	})
	req := httptest.NewRequest(http.MethodPost, "http://127.0.0.1:8001/account/security/passkey/begin", nil)
	req.Host = "127.0.0.1:8001"

	_, err := WebAuthnInstance(ctx, req)
	if err == nil {
		t.Fatal("expected error for IP RP ID")
	}
	if !strings.Contains(err.Error(), "localhost:8001") {
		t.Fatalf("error = %q, want localhost:8001 hint", err)
	}
}
