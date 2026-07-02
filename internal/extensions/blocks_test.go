package extensions

import (
	"testing"

	"github.com/rob121/cannon/extension"
)

func TestMatchBlock(t *testing.T) {
	blocks := []extension.BlockDefinition{
		{ID: "contact-form", Spaces: []string{"footer", "sidebar"}},
		{ID: "newsletter", Spaces: []string{"footer"}},
	}

	if id, ok := MatchBlock(blocks, "footer"); !ok || id != "contact-form" {
		t.Fatalf("footer match: id=%q ok=%v", id, ok)
	}
	if id, ok := MatchBlock(blocks, "sidebar"); !ok || id != "contact-form" {
		t.Fatalf("sidebar match: id=%q ok=%v", id, ok)
	}
	if id, ok := MatchBlock([]extension.BlockDefinition{{ID: "footer"}}, "footer"); !ok || id != "footer" {
		t.Fatalf("id match: id=%q ok=%v", id, ok)
	}
	if id, ok := MatchBlock([]extension.BlockDefinition{{ID: "default", Spaces: nil}}, "any-space"); !ok || id != "default" {
		t.Fatalf("wildcard match: id=%q ok=%v", id, ok)
	}
	if _, ok := MatchBlock(blocks, "header"); ok {
		t.Fatal("expected no match for header")
	}
}

func TestCapabilityPath(t *testing.T) {
	if got := capabilityPath("/block", ""); got != "/block" {
		t.Fatalf("base path: got %q", got)
	}
	if got := capabilityPath("/block", "contact-form"); got != "/block/contact-form" {
		t.Fatalf("item path: got %q", got)
	}
}
