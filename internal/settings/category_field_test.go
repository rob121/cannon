package settings_test

import (
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/rob121/cannon/extension"
	"github.com/rob121/cannon/internal/models"
	"github.com/rob121/cannon/internal/settings"
)

func TestRenderFormCategoryField(t *testing.T) {
	schema := []byte(`{
		"type": "object",
		"properties": {
			"listing_category_id": {
				"type": ["integer", "null"],
				"format": "category",
				"title": "Listing Category",
				"description": "Default category for new items."
			}
		}
	}`)
	uiSchema := []byte(`{
		"type": "VerticalLayout",
		"elements": [{
			"type": "Control",
			"scope": "#/properties/listing_category_id",
			"options": {"format": "category"}
		}]
	}`)
	categories := []models.Category{
		{CategoryID: 1, Name: "News"},
		{CategoryID: 2, Name: "Events"},
	}
	html, err := settings.RenderForm(extension.ConfigurationSection{
		ID:       "content",
		Title:    "Content",
		Schema:   schema,
		UISchema: uiSchema,
		Data:     []byte(`{"listing_category_id":2}`),
	}, "/admin/configuration/global/content", "", &settings.FormRenderContext{Categories: categories})
	if err != nil {
		t.Fatal(err)
	}
	for _, part := range []string{
		`class="form-select admin-form-control admin-config-category"`,
		`name="#/properties/listing_category_id"`,
		`<option value="2" selected>Events</option>`,
		`<option value="1">News</option>`,
	} {
		if !strings.Contains(html, part) {
			t.Fatalf("expected %q in form html:\n%s", part, html)
		}
	}
	if strings.Contains(html, `type="number"`) {
		t.Fatal("expected number input replaced with category select")
	}
}

func TestFormDataFromRequestCategoryField(t *testing.T) {
	schema := []byte(`{"type":"object","properties":{"listing_category_id":{"type":["integer","null"],"format":"category"}}}`)
	uiSchema := []byte(`{"type":"VerticalLayout","elements":[{"type":"Control","scope":"#/properties/listing_category_id","options":{"format":"category"}}]}`)

	form := url.Values{"#/properties/listing_category_id": {"3"}}
	r := &http.Request{Form: form}
	data := settings.FormDataFromRequest(r, schema, uiSchema)
	if got, ok := data["listing_category_id"].(uint64); !ok || got != 3 {
		t.Fatalf("listing_category_id: %+v", data["listing_category_id"])
	}

	form = url.Values{"#/properties/listing_category_id": {""}}
	r = &http.Request{Form: form}
	data = settings.FormDataFromRequest(r, schema, uiSchema)
	if data["listing_category_id"] != nil {
		t.Fatalf("expected nil for empty category: %+v", data["listing_category_id"])
	}
}

func TestRenderCategorySelectHTMLIncludesInactiveSelection(t *testing.T) {
	html := settings.RenderCategorySelectHTML("#/properties/listing_category_id", nil, 9)
	if !strings.Contains(html, `<option value="9" selected>Category #9</option>`) {
		t.Fatalf("expected inactive selected category option: %q", html)
	}
}
