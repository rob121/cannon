package content_test

import (
	"strings"
	"testing"

	"github.com/rob121/cannon/extension"
	"github.com/rob121/cannon/internal/content"
	"github.com/rob121/cannon/internal/models"
	"github.com/rob121/cannon/internal/settings"
)

func TestInjectConfigurationPermissionFieldsBeforeSubmit(t *testing.T) {
	def, ok, err := settings.GlobalDefinition("content")
	if err != nil || !ok {
		t.Fatal(err, ok)
	}
	html, err := settings.RenderForm(extension.ConfigurationSection{
		ID: def.ID, Title: def.Title, Schema: def.Schema, UISchema: def.UISchema,
	}, "/admin/configuration/global/content", "")
	if err != nil {
		t.Fatal(err)
	}
	out := content.InjectConfigurationPermissionFields(html, nil, nil, nil, nil, nil, 0)
	submit := strings.Index(out, `btn-admin-primary`)
	perms := strings.Index(out, "Frontend Permissions")
	if submit < 0 || perms < 0 {
		t.Fatalf("missing markers submit=%d perms=%d", submit, perms)
	}
	if submit < perms {
		t.Fatalf("submit button should come after injected fields; submit=%d perms=%d\nsnippet:\n%s", submit, perms, out[submit:submit+300])
	}
}

func TestInjectConfigurationPermissionFields(t *testing.T) {
	groups := []models.Group{
		{GroupID: 1, Name: "Writers", Status: models.StatusActive},
		{GroupID: 2, Name: "Editors", Status: models.StatusActive},
	}
	profiles := []models.Profile{
		{ProfileID: 5, Name: "Author"},
	}
	form := `<form action="/admin/configuration/global/content"><button type="submit">Save</button></form>`
	out := content.InjectConfigurationPermissionFields(form, groups, profiles, []uint{1}, nil, []uint{2}, 5)
	if !strings.Contains(out, `name="create_group_ids"`) {
		t.Fatal("expected create group field")
	}
	if !strings.Contains(out, `name="author_profile_id"`) {
		t.Fatal("expected author profile field")
	}
	if !strings.Contains(out, `value="5" selected`) {
		t.Fatal("expected author profile selected")
	}
	if !strings.Contains(out, "</form>") {
		t.Fatal("expected form tag preserved")
	}
}
