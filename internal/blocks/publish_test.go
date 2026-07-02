package blocks

import (
	"html/template"
	"strings"
	"testing"
	"time"

	"github.com/rob121/cannon/internal/models"
)

func TestPublishVisible(t *testing.T) {
	now := time.Date(2026, 7, 1, 12, 0, 0, 0, time.UTC)
	future := now.Add(time.Hour)
	past := now.Add(-time.Hour)
	before := now.Add(-2 * time.Hour)
	after := now.Add(2 * time.Hour)

	tests := []struct {
		name     string
		meta     Metadata
		expected bool
	}{
		{name: "open", meta: Metadata{}, expected: true},
		{name: "within window", meta: Metadata{PublishStart: &before, PublishEnd: &after}, expected: true},
		{name: "before start", meta: Metadata{PublishStart: &future}, expected: false},
		{name: "after end", meta: Metadata{PublishEnd: &past}, expected: false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := PublishVisible(tc.meta, now); got != tc.expected {
				t.Fatalf("PublishVisible() = %v, want %v", got, tc.expected)
			}
		})
	}
}

func TestMetadataFromFormValuesSettings(t *testing.T) {
	values := map[string][]string{
		"template_wrapper": {"default/partials/blocks/default.html"},
		"show_name":          {"1"},
		"publish_start":      {"2026-07-01T09:00"},
		"publish_end":        {"2026-07-31T17:00"},
	}
	raw, err := MetadataFromFormValues("html", "hello", values)
	if err != nil {
		t.Fatal(err)
	}
	meta, err := ParseMetadata(raw)
	if err != nil {
		t.Fatal(err)
	}
	if meta.TemplateWrapper != "default/partials/blocks/default.html" {
		t.Fatalf("template_wrapper = %q", meta.TemplateWrapper)
	}
	if !meta.ShowName {
		t.Fatal("show_name should be true")
	}
	if meta.PublishStart == nil || meta.PublishEnd == nil {
		t.Fatalf("publish dates missing: %#v", meta)
	}
}

func TestFinishBlockHTMLShowName(t *testing.T) {
	row := models.Block{Name: "Footer Links", Space: "footer"}
	meta := Metadata{ShowName: true}
	render := func(name string, data map[string]any) (string, error) {
		if name != CardWrapperTemplate {
			t.Fatalf("template = %q, want %q", name, CardWrapperTemplate)
		}
		return `<div class="card-header">` + row.Name + `</div>` + string(data["Body"].(template.HTML)), nil
	}
	got, err := finishBlockHTML(row, meta, "<p>Links</p>", render)
	if err != nil {
		t.Fatal(err)
	}
	if got == "" || !strings.Contains(got, "Footer Links") || !strings.Contains(got, "Links") || !strings.Contains(got, "card-header") {
		t.Fatalf("unexpected html: %q", got)
	}
}

func TestFinishBlockHTMLUsesCardWhenNoWrapper(t *testing.T) {
	row := models.Block{Name: "Sidebar", Space: "sidebar"}
	meta := Metadata{}
	render := func(name string, _ map[string]any) (string, error) {
		if name != CardWrapperTemplate {
			t.Fatalf("template = %q", name)
		}
		return "ok", nil
	}
	got, err := finishBlockHTML(row, meta, "<p>Hi</p>", render)
	if err != nil || got != "ok" {
		t.Fatalf("got %q err %v", got, err)
	}
}

func TestFinishBlockHTMLWrapper(t *testing.T) {
	row := models.Block{Name: "Promo", Space: "sidebar"}
	meta := Metadata{TemplateWrapper: "default/partials/blocks/default.html", ShowName: true}
	render := func(name string, data map[string]any) (string, error) {
		if name != DefaultWrapperTemplate {
			t.Fatalf("template = %q", name)
		}
		if data["Name"] != row.Name || !data["ShowName"].(bool) {
			t.Fatalf("data = %#v", data)
		}
		return `<wrapped>` + string(data["Body"].(template.HTML)) + `</wrapped>`, nil
	}
	got, err := finishBlockHTML(row, meta, "<em>Hi</em>", render)
	if err != nil {
		t.Fatal(err)
	}
	if got != "<wrapped><em>Hi</em></wrapped>" {
		t.Fatalf("got %q", got)
	}
}
