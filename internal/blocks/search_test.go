package blocks

import (
	"context"
	"testing"
)

func TestRenderSearchHorizontalBlock(t *testing.T) {
	rendered := ""
	render := func(name string, data map[string]any) (string, error) {
		if name != SearchHorizontalBlockTemplate {
			t.Fatalf("template = %q", name)
		}
		if data["Action"] != "/content/search" {
			t.Fatalf("action = %#v", data["Action"])
		}
		if data["Placeholder"] != "Find items…" {
			t.Fatalf("placeholder = %#v", data["Placeholder"])
		}
		if data["Button"] != "Go" {
			t.Fatalf("button = %#v", data["Button"])
		}
		if data["Class"] != "header-search" {
			t.Fatalf("class = %#v", data["Class"])
		}
		if data["BlockID"] != uint(7) {
			t.Fatalf("block id = %#v", data["BlockID"])
		}
		rendered = `<form>ok</form>`
		return rendered, nil
	}
	got, err := RenderSearchHorizontalBlock(context.Background(), BlockRow{BlockID: 7}, Metadata{
		SearchPlaceholder: "Find items…",
		SearchButton:      "Go",
		SearchClass:       "header-search",
	}, render)
	if err != nil {
		t.Fatal(err)
	}
	if got != rendered {
		t.Fatalf("got %q", got)
	}
}

func TestRenderSearchVerticalBlockDefaults(t *testing.T) {
	render := func(name string, data map[string]any) (string, error) {
		if name != SearchVerticalBlockTemplate {
			t.Fatalf("template = %q", name)
		}
		if data["Placeholder"] != "Search…" || data["Button"] != "Search" {
			t.Fatalf("defaults = %#v", data)
		}
		return "<form>ok</form>", nil
	}
	got, err := RenderSearchVerticalBlock(context.Background(), BlockRow{BlockID: 1}, Metadata{}, render)
	if err != nil {
		t.Fatal(err)
	}
	if got != "<form>ok</form>" {
		t.Fatalf("got %q", got)
	}
}

func TestRenderSearchBlockNilRenderer(t *testing.T) {
	got, err := RenderSearchHorizontalBlock(context.Background(), BlockRow{BlockID: 1}, Metadata{}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if got != "" {
		t.Fatalf("got %q", got)
	}
}
