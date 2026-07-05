package realtime

import (
	"testing"

	"github.com/centrifugal/centrifuge"
)

func TestBuildStats(t *testing.T) {
	result := centrifuge.PresenceResult{
		Presence: map[string]*centrifuge.ClientInfo{
			"a": {
				UserID:   "anon:abc",
				ChanInfo: []byte(`{"page":"/"}`),
			},
			"b": {
				UserID:   "user:7",
				ChanInfo: []byte(`{"page":"/blog"}`),
			},
			"c": {
				UserID:   "anon:def",
				ChanInfo: []byte(`{"page":"/blog"}`),
			},
		},
	}
	stats := buildStats(result)
	if stats.Online != 3 {
		t.Fatalf("online=%d", stats.Online)
	}
	if stats.Authenticated != 1 {
		t.Fatalf("authenticated=%d", stats.Authenticated)
	}
	if len(stats.Pages) != 2 || stats.Pages[0].Path != "/blog" || stats.Pages[0].Count != 2 {
		t.Fatalf("pages=%+v", stats.Pages)
	}
}
