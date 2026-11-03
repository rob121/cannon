package content

import (
	"context"
	"strings"
	"testing"
)

func TestRichTextToHTMLMarkdown(t *testing.T) {
	html, err := RichTextToHTML(context.Background(), "Hello **world**")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(html, "<strong>world</strong>") {
		t.Fatalf("html = %q", html)
	}
}

func TestRichTextToHTMLSanitizesHTML(t *testing.T) {
	src := "<p>Hello <strong>world</strong></p>"
	html, err := RichTextToHTML(context.Background(), src)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(html, "<strong>world</strong>") {
		t.Fatalf("html = %q", html)
	}
}

func TestRichTextToHTMLStripsScript(t *testing.T) {
	src := `<p>Hi</p><script>alert(1)</script>`
	html, err := RichTextToHTML(context.Background(), src)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(html, "script") {
		t.Fatalf("html = %q", html)
	}
}

func TestSanitizeStoredRichTextLeavesMarkdown(t *testing.T) {
	md := "Hello **world**"
	if got := SanitizeStoredRichText(context.Background(), md); got != md {
		t.Fatalf("got = %q", got)
	}
}

func TestIsHTMLContent(t *testing.T) {
	if IsHTMLContent("plain text") {
		t.Fatal("expected false for plain text")
	}
	if !IsHTMLContent("<p>html</p>") {
		t.Fatal("expected true for html")
	}
}
