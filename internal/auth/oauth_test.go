package auth

import "testing"

func TestSanitizeUsername(t *testing.T) {
	if got := sanitizeUsername("Jane Doe!"); got != "janedoe" {
		t.Fatalf("got %q", got)
	}
	if got := sanitizeUsername(""); got != "" {
		t.Fatalf("got %q", got)
	}
}

func TestParseOAuthScopes(t *testing.T) {
	got := parseOAuthScopes("email, profile openid")
	if len(got) != 3 || got[0] != "email" || got[1] != "profile" || got[2] != "openid" {
		t.Fatalf("got %#v", got)
	}
}

func TestOAuthCallbackPath(t *testing.T) {
	if OAuthCallbackPath("google") != "/auth/oauth/google/callback" {
		t.Fatalf("got %q", OAuthCallbackPath("google"))
	}
}
