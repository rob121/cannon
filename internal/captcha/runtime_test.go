package captcha

import (
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestExpandHTMLDisabledStripsTags(t *testing.T) {
	body := `<form>` + PlaceholderMarkup(CaptchaContextLogin, ProviderAny) + `</form>`
	got, err := ExpandHTML(t.Context(), body)
	if err != nil {
		t.Fatal(err)
	}
	if got != "<form></form>" {
		t.Fatalf("got %q", got)
	}
}

func TestParseTagAttrs(t *testing.T) {
	attrs := parseTagAttrs(`<captcha context="login" provider="any"></captcha>`)
	if attrs["context"] != "login" || attrs["provider"] != "any" {
		t.Fatalf("attrs = %#v", attrs)
	}
	legacy := parseTagAttrs(`<captcha context="comment" type="cloudflare"/>`)
	if legacy["type"] != "cloudflare" {
		t.Fatalf("legacy = %#v", legacy)
	}
}

func TestInjectHeadHTML(t *testing.T) {
	body := "<html><head><title>x</title></head><body></body></html>"
	got := injectHeadHTML(body, []string{`<script src="x"></script>`})
	if !stringsContains(got, `<script src="x"></script>`) || !stringsContains(got, "</head>") {
		t.Fatalf("got %q", got)
	}
}

func TestParseFormPreserveBodyKeepsRequestBody(t *testing.T) {
	req, err := http.NewRequest(http.MethodPost, "/ext/hash/contact/submit", strings.NewReader("captcha_context=form&message=hello"))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	if err := parseFormPreserveBody(req); err != nil {
		t.Fatal(err)
	}
	if got := req.FormValue("message"); got != "hello" {
		t.Fatalf("message = %q", got)
	}
	raw, err := io.ReadAll(req.Body)
	if err != nil {
		t.Fatal(err)
	}
	if string(raw) != "captcha_context=form&message=hello" {
		t.Fatalf("body = %q", string(raw))
	}
}

func stringsContains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(sub) == 0 || indexOf(s, sub) >= 0)
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
