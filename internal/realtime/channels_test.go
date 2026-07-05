package realtime

import "testing"

func TestChannelHelpers(t *testing.T) {
	siteID := "demo"
	presence := PresenceChannel(siteID)
	if presence != "site:demo:presence" {
		t.Fatalf("unexpected presence channel: %q", presence)
	}
	analytics := AnalyticsChannel(siteID)
	if analytics != "site:demo:analytics" {
		t.Fatalf("unexpected analytics channel: %q", analytics)
	}
	gotSite, ok := SiteIDFromChannel(presence)
	if !ok || gotSite != siteID {
		t.Fatalf("site id from channel: got %q ok=%v", gotSite, ok)
	}
	gotSite, kind, ok := ChannelKind(analytics)
	if !ok || gotSite != siteID || kind != "analytics" {
		t.Fatalf("channel kind: site=%q kind=%q ok=%v", gotSite, kind, ok)
	}
}

func TestEndpointURL(t *testing.T) {
	got := EndpointURL("https", "example.com")
	want := "wss://example.com/connection/websocket"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}
