package auth

import "testing"

func TestProviderConfigured(t *testing.T) {
	if !ProviderConfigured("local", `{}`) {
		t.Fatal("local should always be configured when active")
	}
	if ProviderConfigured("github", `{}`) {
		t.Fatal("github should require credentials")
	}
	cfg := `{"client_key":"id","secret":"sec"}`
	if !ProviderConfigured("github", cfg) {
		t.Fatal("github should be configured with client key and secret (callback is optional)")
	}
	if !ProviderConfigured("google", cfg) {
		t.Fatal("google should be configured without callback_url")
	}
	if ProviderConfigured("github", `{"client_key":"id"}`) {
		t.Fatal("github should require secret")
	}
}

func TestSuggestedCallbackURL(t *testing.T) {
	got := SuggestedCallbackURL("https://floridarangers.com", "google")
	want := "https://floridarangers.com/auth/oauth/google/callback"
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
	if SuggestedCallbackURL("", "google") != "/auth/oauth/google/callback" {
		t.Fatalf("path-only fallback = %q", SuggestedCallbackURL("", "google"))
	}
}

func TestLoginProvidersFiltersInactiveConfig(t *testing.T) {
	providers := []LoginProvider{
		{Name: "local", Label: "Local", Kind: "local"},
		{Name: "github", Label: "Github", Kind: "oauth"},
	}
	if !LocalLoginEnabled(providers) {
		t.Fatal("expected local enabled")
	}
	oauth := OAuthProviders(providers)
	if len(oauth) != 1 || oauth[0].Name != "github" {
		t.Fatalf("oauth providers = %#v", oauth)
	}
}

func TestOAuthBeginPath(t *testing.T) {
	if OAuthBeginPath("google") != "/auth/oauth/google" {
		t.Fatalf("got %q", OAuthBeginPath("google"))
	}
	if OAuthBeginPath("local") != "" {
		t.Fatal("local should not have oauth path")
	}
}
