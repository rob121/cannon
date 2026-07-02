package extensions

import (
	"encoding/json"
	"testing"
)

func TestParseHelpListStrings(t *testing.T) {
	raw := json.RawMessage(`["/help/overview","/help/how-to-config"]`)
	entries, err := parseHelpList(raw)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}
	if entries[0].Path != "/help/overview" || entries[0].Title != "Overview" {
		t.Fatalf("unexpected first entry: %#v", entries[0])
	}
	if entries[1].Title != "How to config" {
		t.Fatalf("unexpected title: %q", entries[1].Title)
	}
}

func TestParseHelpListObjects(t *testing.T) {
	raw := json.RawMessage(`[{"path":"/help/overview","title":"Getting Started"}]`)
	entries, err := parseHelpList(raw)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].Title != "Getting Started" {
		t.Fatalf("unexpected entries: %#v", entries)
	}
}

func TestHelpArticleURL(t *testing.T) {
	got := HelpArticleURL("blog", "/help/overview")
	want := "/admin/help/extensions/blog/help/overview"
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}
