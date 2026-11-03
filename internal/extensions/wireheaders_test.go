package extensions

import (
	"net/http"
	"testing"
)

func TestFilterWireHeadersStripsCookie(t *testing.T) {
	h := http.Header{
		"Cookie":        {"session=secret"},
		"Content-Type":  {"application/json"},
		"Authorization": {"Bearer token"},
	}
	out := FilterWireHeaders(h)
	if _, ok := out["Cookie"]; ok {
		t.Fatal("cookie should be stripped")
	}
	if out["Content-Type"][0] != "application/json" {
		t.Fatalf("content-type = %v", out["Content-Type"])
	}
}
