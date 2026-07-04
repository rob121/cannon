package captcha

import "testing"

func TestPlaceholderMarkup(t *testing.T) {
	got := PlaceholderMarkup("login", ProviderAny)
	want := `<captcha context="login" provider="any"></captcha>`
	if got != want {
		t.Fatalf("got %q", got)
	}
}

func TestPlaceholderMarkupDefaults(t *testing.T) {
	got := PlaceholderMarkup("", "")
	if got != `<captcha context="form" provider="any"></captcha>` {
		t.Fatalf("got %q", got)
	}
}
