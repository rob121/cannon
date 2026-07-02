package settings_test

import (
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/rob121/cannon/extension"
	"github.com/rob121/cannon/internal/settings"
)

func TestRenderFormBooleanToggle(t *testing.T) {
	def, ok, err := settings.GlobalDefinition("general")
	if err != nil || !ok {
		t.Fatal(err, ok)
	}
	html, err := settings.RenderForm(extension.ConfigurationSection{
		ID:       def.ID,
		Title:    def.Title,
		Schema:   def.Schema,
		UISchema: def.UISchema,
		Data:     []byte(`{"debug_template_spaces":true,"site_offline":false}`),
	}, "/admin/configuration/global/general", "")
	if err != nil {
		t.Fatal(err)
	}
	for _, part := range []string{
		`admin-form-toggle-input`,
		`admin-form-toggle-track`,
		`name="#/properties/debug_template_spaces"`,
		`name="#/properties/site_offline"`,
	} {
		if !strings.Contains(html, part) {
			t.Fatalf("expected %q in form html", part)
		}
	}
	if strings.Contains(html, `type="boolean"`) {
		t.Fatalf("expected boolean inputs replaced with toggles: %q", html)
	}
	if !strings.Contains(html, `name="#/properties/debug_template_spaces" value="true" checked`) {
		t.Fatal("expected debug_template_spaces toggle checked")
	}
	if strings.Contains(html, `name="#/properties/site_offline" value="true" checked`) {
		t.Fatal("expected site_offline toggle unchecked")
	}
}

func TestRenderFormInjectsCSRF(t *testing.T) {
	def, ok, err := settings.GlobalDefinition("general")
	if err != nil || !ok {
		t.Fatal(err, ok)
	}
	html, err := settings.RenderForm(extension.ConfigurationSection{
		ID: def.ID, Title: def.Title, Schema: def.Schema, UISchema: def.UISchema,
	}, "/admin/configuration/global/general", "abc123")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(html, `name="_csrf" value="abc123"`) {
		t.Fatalf("expected csrf field in form html: %q", html)
	}
}

func TestFormDataFromRequestBooleanDefaults(t *testing.T) {
	def, ok, err := settings.GlobalDefinition("general")
	if err != nil || !ok {
		t.Fatal(err, ok)
	}
	form := url.Values{
		"#/properties/debug_template_spaces": {"true"},
	}
	r := &http.Request{Form: form}
	data := settings.FormDataFromRequest(r, def.Schema)
	if !settings.Bool(data, "debug_template_spaces") {
		t.Fatalf("debug_template_spaces: %+v", data)
	}
	if settings.Bool(data, "site_offline") {
		t.Fatalf("site_offline should default false: %+v", data)
	}
}

func TestApplyBooleanFormDefaults(t *testing.T) {
	schema := []byte(`{"type":"object","properties":{"enabled":{"type":"boolean"}}}`)
	data := settings.ApplyBooleanFormDefaults(schema, map[string]any{}, url.Values{})
	if settings.Bool(data, "enabled") {
		t.Fatalf("expected false default: %+v", data)
	}
}
