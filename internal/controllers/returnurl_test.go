package controllers

import (
	"net/http/httptest"
	"testing"

	"github.com/rob121/cannon/internal/config"
)

func TestSafeRelativePath(t *testing.T) {
	cases := []struct {
		in   string
		ok   bool
		want string
	}{
		{"/dashboard", true, "/dashboard"},
		{"dashboard", false, ""},
		{"//evil.com", false, ""},
		{"/https://evil.com", false, ""},
		{"http://evil.com", false, ""},
		{"/account/verify/token", true, "/account/verify/token"},
	}
	for _, tc := range cases {
		got, ok := SafeRelativePath(tc.in)
		if ok != tc.ok || (tc.ok && got != tc.want) {
			t.Fatalf("SafeRelativePath(%q): got %q %v want %q %v", tc.in, got, ok, tc.want, tc.ok)
		}
	}
}

func TestEncodeDecodeReturn(t *testing.T) {
	site := &config.SiteConfig{ID: "demo", TmpDir: "/tmp/demo", Host: "127.0.0.1:8001"}
	encoded, err := EncodeReturn(site, "/members")
	if err != nil {
		t.Fatal(err)
	}
	path, err := DecodeReturn(site, encoded)
	if err != nil || path != "/members" {
		t.Fatalf("DecodeReturn: got %q err=%v", path, err)
	}
	req := httptest.NewRequest("GET", "/login?return="+encoded, nil)
	if got := ReturnParam(site, req, "/"); got != "/members" {
		t.Fatalf("ReturnParam: got %q", got)
	}
}
