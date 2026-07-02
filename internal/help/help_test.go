package help

import (
	"strings"
	"testing"
)

func TestIndex(t *testing.T) {
	sections, err := Index()
	if err != nil {
		t.Fatal(err)
	}
	if len(sections) < 2 {
		t.Fatalf("expected at least 2 folders, got %d", len(sections))
	}
	if sections[0].Folder != "admin" && sections[0].Folder != "getting-started" {
		t.Fatalf("unexpected first folder: %q", sections[0].Folder)
	}
}

func TestFetch(t *testing.T) {
	md, err := Fetch("getting-started", "overview")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(md, "Cannon") {
		t.Fatalf("expected overview content")
	}
}

func TestFetchFrontendTemplatesIncludesTemplateJSON(t *testing.T) {
	md, err := Fetch("templates", "frontend-templates")
	if err != nil {
		t.Fatal(err)
	}
	for _, phrase := range []string{"template.json", "Frontend Theme", "assets/", "Configuration → General"} {
		if !strings.Contains(md, phrase) {
			t.Fatalf("expected frontend-templates help to mention %q", phrase)
		}
	}
}

func TestArticleURL(t *testing.T) {
	got := ArticleURL("admin", "admin-basics")
	want := "/admin/help/admin/admin-basics"
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}
