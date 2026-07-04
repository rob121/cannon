package settings_test

import (
	"encoding/json"
	"testing"

	"github.com/rob121/cannon/extension"
	"github.com/rob121/cannon/internal/settings"
)

func TestMergeLegacySEOMetaSection(t *testing.T) {
	section := extension.ConfigurationSection{
		ID: "seo",
		Data: []byte(`{"robots_txt":"User-agent: *"}`),
	}
	general := map[string]any{
		"site_meta_description": "Legacy description",
		"site_name":             "Example",
	}
	merged := settings.MergeLegacySEOMetaSection(section, general)
	var data map[string]any
	if err := json.Unmarshal(merged.Data, &data); err != nil {
		t.Fatal(err)
	}
	if data["site_meta_description"] != "Legacy description" {
		t.Fatalf("expected legacy meta description, got %+v", data)
	}
	if _, ok := data["site_name"]; ok {
		t.Fatal("site_name should not be copied into seo section")
	}
	if data["robots_txt"] != "User-agent: *" {
		t.Fatalf("existing seo values should be preserved: %+v", data)
	}
}

func TestGlobalSEODefinitionHasMetaTags(t *testing.T) {
	def, ok, err := settings.GlobalDefinition("seo")
	if err != nil || !ok {
		t.Fatal(err, ok)
	}
	for _, key := range []string{"site_meta_description", "robots_txt"} {
		if !jsonContainsKey(def.Schema, key) {
			t.Fatalf("expected %q in seo schema", key)
		}
	}
	def, ok, err = settings.GlobalDefinition("general")
	if err != nil || !ok {
		t.Fatal(err, ok)
	}
	if jsonContainsKey(def.Schema, "site_meta_description") {
		t.Fatal("site_meta_description should not remain in general schema")
	}
}

func jsonContainsKey(raw json.RawMessage, key string) bool {
	var doc struct {
		Properties map[string]any `json:"properties"`
	}
	if err := json.Unmarshal(raw, &doc); err != nil {
		return false
	}
	_, ok := doc.Properties[key]
	return ok
}
