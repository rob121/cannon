package settings_test

import (
	"fmt"
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
	}, "/admin/configuration/global/general", "", nil)
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
	if !strings.Contains(html, `name="#/properties/captcha_enabled"`) || !strings.Contains(html, `admin-form-toggle-input`) {
		t.Fatal("expected captcha_enabled toggle in grouped section")
	}
	if !strings.Contains(html, `name="#/properties/captcha_skip_authenticated"`) {
		t.Fatal("expected captcha_skip_authenticated toggle in grouped section")
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
	}, "/admin/configuration/global/general", "abc123", nil)
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
	data := settings.FormDataFromRequest(r, def.Schema, def.UISchema)
	if !settings.Bool(data, "debug_template_spaces") {
		t.Fatalf("debug_template_spaces: %+v", data)
	}
	if settings.Bool(data, "site_offline") {
		t.Fatalf("site_offline should default false: %+v", data)
	}
}

func TestGlobalMediaDefinition(t *testing.T) {
	def, ok, err := settings.GlobalDefinition("media")
	if err != nil || !ok {
		t.Fatal(err, ok)
	}
	if def.Title != "Media" {
		t.Fatalf("title = %q", def.Title)
	}
	if !strings.Contains(string(def.Schema), "max_file_size_mb") {
		t.Fatal("expected max_file_size_mb in schema")
	}
	if !strings.Contains(string(def.Schema), "approved_extensions") {
		t.Fatal("expected approved_extensions in schema")
	}
}

func TestGlobalMailDefinition(t *testing.T) {
	def, ok, err := settings.GlobalDefinition("mail")
	if err != nil || !ok {
		t.Fatal(err, ok)
	}
	if def.Title != "Mail" {
		t.Fatalf("title = %q", def.Title)
	}
	if !strings.Contains(string(def.Schema), "use_html") {
		t.Fatal("expected use_html in mail schema")
	}
	if !strings.Contains(string(def.Schema), "html_template") {
		t.Fatal("expected html_template in mail schema")
	}
}

func TestApplyBooleanFormDefaults(t *testing.T) {
	schema := []byte(`{"type":"object","properties":{"enabled":{"type":"boolean"}}}`)
	data := settings.ApplyBooleanFormDefaults(schema, map[string]any{}, url.Values{})
	if settings.Bool(data, "enabled") {
		t.Fatalf("expected false default: %+v", data)
	}
}

func TestRenderFormSelectFieldsSubmitAndDisplay(t *testing.T) {
	schema := []byte(`{"type":"object","properties":{"frontend_theme":{"type":"string","title":"Frontend Theme","enum":["default","fr"]},"log_level":{"type":"string","enum":["info","debug"]},"default_list_limit":{"type":"integer","enum":[25,50]}}}`)
	uiSchema := []byte(`{"type":"VerticalLayout","elements":[{"type":"Control","scope":"#/properties/frontend_theme"},{"type":"Control","scope":"#/properties/log_level"},{"type":"Control","scope":"#/properties/default_list_limit"}]}`)
	html, err := settings.RenderForm(extension.ConfigurationSection{
		ID:       "general",
		Title:    "General",
		Schema:   schema,
		UISchema: uiSchema,
		Data:     []byte(`{"frontend_theme":"fr","log_level":"debug","default_list_limit":50}`),
	}, "/admin/configuration/global/general", "", nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, part := range []string{
		`name="#/properties/frontend_theme"`,
		`name="#/properties/log_level"`,
		`name="#/properties/default_list_limit"`,
		`<option value="fr" selected>fr</option>`,
		`<option value="debug" selected>debug</option>`,
		`<option value="50" selected>50</option>`,
	} {
		if !strings.Contains(html, part) {
			t.Fatalf("expected %q in form html", part)
		}
	}
}

func TestRenderFormTextareaFields(t *testing.T) {
	def, ok, err := settings.GlobalDefinition("seo")
	if err != nil || !ok {
		t.Fatal(err, ok)
	}
	html, err := settings.RenderForm(extension.ConfigurationSection{
		ID: def.ID, Title: def.Title, Schema: def.Schema, UISchema: def.UISchema,
		Data: []byte(`{"site_meta_description":"Line one\nLine two","site_meta_keywords":"a, b"}`),
	}, "/admin/configuration/global/seo", "", nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, part := range []string{
		`<textarea class="form-input admin-form-control admin-form-textarea" id="#/properties/site_meta_description"`,
		`Line one`,
		`<textarea class="form-input admin-form-control admin-form-textarea" id="#/properties/site_meta_keywords"`,
		`<textarea class="form-input admin-form-control admin-form-textarea" id="#/properties/site_head_extra"`,
		`Default Meta Tags`,
	} {
		if !strings.Contains(html, part) {
			t.Fatalf("expected %q in form html:\n%s", part, html)
		}
	}
	if strings.Contains(html, `id="#/properties/site_meta_description" name="#/properties/site_meta_description" type="text"`) {
		t.Fatal("expected meta description as textarea, not text input")
	}
}

func TestFormDataFromRequestEnumFields(t *testing.T) {
	def, ok, err := settings.GlobalDefinition("general")
	if err != nil || !ok {
		t.Fatal(err, ok)
	}
	form := url.Values{
		"#/properties/frontend_theme":     {"fr"},
		"#/properties/admin_theme":        {"admin"},
		"#/properties/log_level":          {"debug"},
		"#/properties/default_list_limit": {"50"},
	}
	r := &http.Request{Form: form}
	data := settings.FormDataFromRequest(r, def.Schema, def.UISchema)
	if got := fmt.Sprint(data["frontend_theme"]); got != "fr" {
		t.Fatalf("frontend_theme: got %q data=%+v", got, data)
	}
	if got := fmt.Sprint(data["log_level"]); got != "debug" {
		t.Fatalf("log_level: got %q data=%+v", got, data)
	}
	if got, _ := data["default_list_limit"].(int); got != 50 {
		if gotF, ok := data["default_list_limit"].(float64); !ok || int(gotF) != 50 {
			t.Fatalf("default_list_limit: %+v", data["default_list_limit"])
		}
	}
}
