package extension

import "testing"

func TestPublicRequestURL(t *testing.T) {
	req := WireRequest{
		URL: "/jobs/example",
		Header: map[string][]string{
			"Host": {"127.0.0.1:8001"},
		},
	}
	got := PublicRequestURL(req, "/jobs/example")
	if got != "http://127.0.0.1:8001/jobs/example" {
		t.Fatalf("got %q", got)
	}
}

func TestPublicRequestURLUsesSiteBase(t *testing.T) {
	req := WireRequest{
		URL: "/jobs/example",
		Settings: map[string]string{
			SettingSitePublicBase: "https://camp.example.org",
		},
	}
	got := PublicRequestURL(req, "/jobs/example")
	if got != "https://camp.example.org/jobs/example" {
		t.Fatalf("got %q", got)
	}
}
