package richtext

import (
	"context"
	"encoding/json"
	"testing"
)

func TestParseToolbarFiltersUnknownItems(t *testing.T) {
	out := parseToolbar("bold, notARealButton, italic", toolbarItemAllow, defaultIntroToolbar)
	if len(out) != 2 {
		t.Fatalf("expected 2 items, got %v", out)
	}
}

func TestClientConfigJSONFallback(t *testing.T) {
	raw := ClientConfigJSON(context.Background())
	var cfg ClientConfig
	if err := json.Unmarshal([]byte(raw), &cfg); err != nil {
		t.Fatal(err)
	}
	if cfg.EnterKeyMode != EnterKeyParagraph || len(cfg.IntroToolbar) == 0 {
		t.Fatalf("unexpected fallback config: %+v", cfg)
	}
}
