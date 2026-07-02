package csrf

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestIsMutating(t *testing.T) {
	if !IsMutating("POST") {
		t.Fatal("POST should be mutating")
	}
	if IsMutating("GET") {
		t.Fatal("GET should not be mutating")
	}
}

func TestValid(t *testing.T) {
	if Valid("abc", "abc") != true {
		t.Fatal("matching tokens should validate")
	}
	if Valid("abc", "abd") {
		t.Fatal("different tokens should not validate")
	}
	if Valid("", "abc") || Valid("abc", "") {
		t.Fatal("empty tokens should not validate")
	}
}

func TestSubmittedToken(t *testing.T) {
	r := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("_csrf=from-form"))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	if got := SubmittedToken(r); got != "from-form" {
		t.Fatalf("form token = %q, want from-form", got)
	}

	r = httptest.NewRequest(http.MethodPost, "/", nil)
	r.Header.Set(HeaderName, "from-header")
	if got := SubmittedToken(r); got != "from-header" {
		t.Fatalf("header token = %q, want from-header", got)
	}

	body := strings.NewReader("--bound\r\nContent-Disposition: form-data; name=\"_csrf\"\r\n\r\nmultipart-token\r\n--bound\r\nContent-Disposition: form-data; name=\"file\"; filename=\"a.txt\"\r\nContent-Type: text/plain\r\n\r\nhello\r\n--bound--\r\n")
	r = httptest.NewRequest(http.MethodPost, "/", body)
	r.Header.Set("Content-Type", "multipart/form-data; boundary=bound")
	if got := SubmittedToken(r); got != "multipart-token" {
		t.Fatalf("multipart token = %q, want multipart-token", got)
	}
}

func TestHiddenFieldEscapes(t *testing.T) {
	got := string(HiddenField(`a"b`))
	if !strings.Contains(got, `value="a&#34;b"`) {
		t.Fatalf("unexpected field html: %s", got)
	}
}
