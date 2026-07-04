package mfa

import (
	"context"
	"strings"
	"testing"

	"github.com/rob121/cannon/internal/config"
	"github.com/rob121/cannon/internal/sites"
)

func TestValidateTOTPCode(t *testing.T) {
	key, err := GenerateTOTPSecret()
	if err != nil {
		t.Fatal(err)
	}
	// pquerna/otp test helper - generate valid code via totp.Generate
	if ValidateTOTPCode(key, "000000") {
		t.Fatal("expected invalid code to fail")
	}
}

func TestTOTPProvisioning(t *testing.T) {
	ctx := sites.WithContext(context.Background(), &config.SiteConfig{
		ID:   "totp-test",
		Name: "Example Site",
	})
	secret, err := GenerateTOTPSecret()
	if err != nil {
		t.Fatal(err)
	}
	prov, err := TOTPProvisioning(ctx, "alice", secret)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(prov.URI, "otpauth://totp/") {
		t.Fatalf("uri: %q", prov.URI)
	}
	if !strings.Contains(prov.URI, secret) {
		t.Fatalf("uri missing secret: %q", prov.URI)
	}
	if !strings.HasPrefix(prov.QRPNGDataURI, "data:image/png;base64,") {
		t.Fatalf("qr data uri prefix missing")
	}
}

func TestSessionUint(t *testing.T) {
	if id, ok := sessionUint(float64(42)); !ok || id != 42 {
		t.Fatalf("float64: got %d ok=%v", id, ok)
	}
	if id, ok := sessionUint(uint(7)); !ok || id != 7 {
		t.Fatalf("uint: got %d ok=%v", id, ok)
	}
}
