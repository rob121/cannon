package extensions

import "testing"

func TestCapabilitiesSummaryUnavailable(t *testing.T) {
	m := &Manager{runtimes: map[string]*Runtime{}}
	summary := m.CapabilitiesSummary("missing")
	if summary.Available {
		t.Fatal("expected unavailable summary")
	}
}

func TestCapabilityItems(t *testing.T) {
	items := capabilityItems(Capabilities{
		Admin: "admin_handler",
		Help:  "/help",
	})
	enabled := 0
	for _, item := range items {
		if item.Key == "admin" {
			if !item.Enabled || item.Handler != "admin_handler" {
				t.Fatalf("unexpected admin item: %#v", item)
			}
		}
		if item.Key == "help" {
			if !item.Enabled || item.Handler != "help" {
				t.Fatalf("unexpected help item: %#v", item)
			}
		}
		if item.Enabled {
			enabled++
		}
	}
	if enabled != 2 {
		t.Fatalf("expected 2 enabled items, got %d", enabled)
	}
	if len(items) != len(capabilityCatalog) {
		t.Fatalf("expected %d catalog items, got %d", len(capabilityCatalog), len(items))
	}
}
