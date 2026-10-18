package media

import "testing"

func TestParseImageStyles(t *testing.T) {
	got, err := parseImageStyles([]any{
		map[string]any{"key": "hero", "width": float64(800), "height": float64(0), "mode": "fit"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].Key != "hero" || got[0].Width != 800 {
		t.Fatalf("got %#v", got)
	}
	def, err := parseImageStyles(nil)
	if err != nil || len(def) != 3 {
		t.Fatalf("defaults: %#v err=%v", def, err)
	}
}
