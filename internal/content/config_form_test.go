package content_test

import (
	"strings"
	"testing"

	"github.com/rob121/cannon/extension"
	"github.com/rob121/cannon/internal/content"
	"github.com/rob121/cannon/internal/models"
	"github.com/rob121/cannon/internal/settings"
)

func TestInjectConfigurationAuthorProfileFieldBeforeSubmit(t *testing.T) {
	def, ok, err := settings.GlobalDefinition("content")
	if err != nil || !ok {
		t.Fatal(err, ok)
	}
	html, err := settings.RenderForm(extension.ConfigurationSection{
		ID: def.ID, Title: def.Title, Schema: def.Schema, UISchema: def.UISchema,
	}, "/admin/configuration/global/content", "", nil)
	if err != nil {
		t.Fatal(err)
	}
	out := content.InjectConfigurationAuthorProfileField(html, nil, 0)
	submit := strings.Index(out, `btn-admin-primary`)
	profile := strings.Index(out, "Author Profiles")
	if submit < 0 || profile < 0 {
		t.Fatalf("missing markers submit=%d profile=%d", submit, profile)
	}
	if submit < profile {
		t.Fatalf("submit button should come after injected fields; submit=%d profile=%d", submit, profile)
	}
}

func TestInjectConfigurationAuthorProfileField(t *testing.T) {
	profiles := []models.Profile{
		{ProfileID: 5, Name: "Author"},
	}
	form := `<form action="/admin/configuration/global/content"><button type="submit">Save</button></form>`
	out := content.InjectConfigurationAuthorProfileField(form, profiles, 5)
	if strings.Contains(out, `create_group_ids`) {
		t.Fatal("frontend permission groups should not be injected")
	}
	if !strings.Contains(out, `name="author_profile_id"`) {
		t.Fatal("expected author profile field")
	}
	if !strings.Contains(out, `value="5" selected`) {
		t.Fatal("expected author profile selected")
	}
}
