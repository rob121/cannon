package settings_test

import (
	"strings"
	"testing"

	"github.com/rob121/cannon/extension"
	"github.com/rob121/cannon/internal/settings"
)

func TestEmbeddedGlobalDefinitions(t *testing.T) {
	defs, err := settings.GlobalDefinitions()
	if err != nil {
		t.Fatal(err)
	}
	if len(defs) < 2 {
		t.Fatalf("expected embedded global sections, got %d", len(defs))
	}
	def, ok, err := settings.GlobalDefinition("mail")
	if err != nil {
		t.Fatal(err)
	}
	if !ok || def.ID != "mail" || def.Title != "Mail" {
		t.Fatalf("mail definition: %+v ok=%v", def, ok)
	}
}

func TestFindSection(t *testing.T) {
	doc := extension.ConfigurationDocument{Sections: []extension.ConfigurationSection{
		{ID: "general", Title: "General"},
		{ID: "mail", Title: "Mail"},
	}}
	section, ok := settings.FindSection(doc, "mail")
	if !ok || section.ID != "mail" {
		t.Fatalf("section: %+v ok=%v", section, ok)
	}
	first, ok := settings.FindSection(doc, "")
	if !ok || first.ID != "general" {
		t.Fatalf("first section: %+v", first)
	}
}

func TestRenderFormUsesAdminClasses(t *testing.T) {
	html, err := settings.RenderForm(extension.ConfigurationSection{
		ID:       "general",
		Title:    "General",
		Schema:   []byte(`{"type":"object","properties":{"site_name":{"type":"string","title":"Site Name"}}}`),
		UISchema: []byte(`{"type":"VerticalLayout","elements":[{"type":"Control","scope":"#/properties/site_name"}]}`),
	}, "/admin/configuration/global/general", "", nil)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(html, "admin-form-control") || !strings.Contains(html, "admin-form-label") || !strings.Contains(html, "btn-admin-primary") {
		t.Fatalf("expected admin classes in form html: %q", html)
	}
}

func TestRenderFormMailDoesNotShowGeneralTitle(t *testing.T) {
	def, ok, err := settings.GlobalDefinition("mail")
	if err != nil || !ok {
		t.Fatal(err, ok)
	}
	html, err := settings.RenderForm(extension.ConfigurationSection{
		ID: def.ID, Title: def.Title, Schema: def.Schema, UISchema: def.UISchema,
	}, "/admin/configuration/global/mail", "", nil)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(html, ">General<") {
		t.Fatalf("mail form should not contain General heading: %q", html)
	}
}

func TestRenderFormMailHTMLTemplatePlaceholder(t *testing.T) {
	def, ok, err := settings.GlobalDefinition("mail")
	if err != nil || !ok {
		t.Fatal(err, ok)
	}
	html, err := settings.RenderForm(extension.ConfigurationSection{
		ID: def.ID, Title: def.Title, Schema: def.Schema, UISchema: def.UISchema,
		Data: []byte(`{}`),
	}, "/admin/configuration/global/mail", "", nil)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(html, `name="#/properties/html_template"`) {
		t.Fatalf("expected html_template field: %q", html)
	}
	if !strings.Contains(html, `placeholder="default/mail/default.html"`) {
		t.Fatalf("expected default mail template placeholder: %q", html)
	}
}

func TestRenderFormGeneralHTML(t *testing.T) {
	def, ok, err := settings.GlobalDefinition("general")
	if err != nil || !ok {
		t.Fatal(err, ok)
	}
	html, err := settings.RenderForm(extension.ConfigurationSection{
		ID: def.ID, Title: def.Title, Schema: def.Schema, UISchema: def.UISchema,
	}, "/admin/configuration/global/general", "", nil)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(html, "<h1") {
		t.Fatalf("unexpected h1 in form html: %q", html)
	}
}
