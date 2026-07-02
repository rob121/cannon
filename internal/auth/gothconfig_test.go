package auth

import (
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
	"testing"
)

func TestDefaultConfigGoogle(t *testing.T) {
	cfg := DefaultConfig("google")
	want := []string{"client_key", "secret", "callback_url", "scopes"}
	if len(cfg) != len(want) {
		t.Fatalf("got %d keys, want %d", len(cfg), len(want))
	}
	for _, k := range want {
		if _, ok := cfg[k]; !ok {
			t.Fatalf("missing key %q", k)
		}
		if cfg[k] != "" {
			t.Fatalf("expected empty default for %q", k)
		}
	}
}

func TestDefaultConfigLocal(t *testing.T) {
	cfg := DefaultConfig("local")
	if cfg["type"] != "bcrypt" {
		t.Fatalf("got type %q, want bcrypt", cfg["type"])
	}
}

func TestMergedConfigPreservesStored(t *testing.T) {
	stored := `{"client_key":"abc","secret":"xyz"}`
	merged := MergedConfig("google", stored)
	if merged["client_key"] != "abc" {
		t.Fatalf("client_key = %q", merged["client_key"])
	}
	if merged["callback_url"] != "" {
		t.Fatalf("expected empty callback_url default")
	}
}

func TestConfigFormFieldsAuth0(t *testing.T) {
	fields := ConfigFormFields("auth0", `{}`)
	keys := map[string]bool{}
	for _, f := range fields {
		keys[f.Key] = true
	}
	for _, want := range []string{"client_key", "secret", "callback_url", "domain", "scopes"} {
		if !keys[want] {
			t.Fatalf("missing field %q", want)
		}
	}
}

func TestDefaultConfigJSONValid(t *testing.T) {
	for _, name := range DefaultProviders {
		raw := DefaultConfigJSON(name)
		var m map[string]string
		if err := json.Unmarshal([]byte(raw), &m); err != nil {
			t.Fatalf("provider %q: invalid json: %v", name, err)
		}
		if len(m) == 0 {
			t.Fatalf("provider %q: empty config", name)
		}
	}
	if raw := DefaultConfigJSON("local"); raw != `{"type":"bcrypt"}` {
		t.Fatalf("local default = %s", raw)
	}
}

func TestConfigFromForm(t *testing.T) {
	form := url.Values{}
	form.Set("cfg_client_key", "id-1")
	form.Set("cfg_secret", "sec-1")
	form.Set("cfg_callback_url", "http://localhost/callback")
	r, err := http.NewRequest(http.MethodPost, "/", strings.NewReader(form.Encode()))
	if err != nil {
		t.Fatal(err)
	}
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	if err := r.ParseForm(); err != nil {
		t.Fatal(err)
	}
	raw := ConfigFromForm(r, "google")
	var cfg map[string]string
	if err := json.Unmarshal([]byte(raw), &cfg); err != nil {
		t.Fatal(err)
	}
	if cfg["client_key"] != "id-1" || cfg["secret"] != "sec-1" {
		t.Fatalf("unexpected cfg: %#v", cfg)
	}
}
