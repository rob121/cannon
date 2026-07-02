package extensions

import "testing"

func TestMetaUpdateURL(t *testing.T) {
	meta := Meta{
		Version:       "0.1.0",
		UpdateURLBase: "https://github.com/rob121/cannon-extension-contact/releases/download",
	}
	got := meta.UpdateURL()
	want := "https://github.com/rob121/cannon-extension-contact/releases/download/0.1.0"
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestMetaUpdateURLEmpty(t *testing.T) {
	if got := (Meta{Version: "1.0.0"}).UpdateURL(); got != "" {
		t.Fatalf("expected empty update url, got %q", got)
	}
}

func TestMetaSummaryUnavailable(t *testing.T) {
	m := &Manager{runtimes: map[string]*Runtime{}}
	summary := m.MetaSummary("missing")
	if summary.Available {
		t.Fatal("expected unavailable meta summary")
	}
}
