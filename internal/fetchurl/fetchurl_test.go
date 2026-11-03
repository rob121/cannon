package fetchurl

import (
	"errors"
	"testing"
)

func TestValidateHTTPURLBlocksPrivateHost(t *testing.T) {
	_, err := ValidateHTTPURL("http://127.0.0.1/asset", Options{AllowHTTP: true})
	if !errors.Is(err, ErrBlockedURL) {
		t.Fatalf("err = %v", err)
	}
}

func TestValidateHTTPURLBlocksLocalhostName(t *testing.T) {
	_, err := ValidateHTTPURL("https://localhost/x", Options{})
	if !errors.Is(err, ErrBlockedURL) {
		t.Fatalf("err = %v", err)
	}
}

func TestValidateHTTPURLRequiresHTTPSByDefault(t *testing.T) {
	_, err := ValidateHTTPURL("http://example.com/x", Options{})
	if err == nil || errors.Is(err, ErrBlockedURL) {
		t.Fatalf("expected http rejection, err = %v", err)
	}
}

func TestValidateGitRemoteURLBlocksSSHLocalhost(t *testing.T) {
	_, err := ValidateGitRemoteURL("git@127.0.0.1:repo/theme.git")
	if !errors.Is(err, ErrBlockedURL) {
		t.Fatalf("err = %v", err)
	}
}

func TestValidateGitRemoteURLAllowsGitHubHTTPS(t *testing.T) {
	got, err := ValidateGitRemoteURL("https://github.com/example/cannon-theme.git")
	if err != nil {
		t.Fatal(err)
	}
	if got == "" {
		t.Fatal("expected URL")
	}
}
