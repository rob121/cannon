package content

import (
	"strings"
	"testing"
)

func TestRichTextToHTMLMarkdown(t *testing.T) {
	html, err := RichTextToHTML("Hello **world**")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(html, "<strong>world</strong>") {
		t.Fatalf("html = %q", html)
	}
}

func TestRichTextToHTMLPassthrough(t *testing.T) {
	src := "<p>Hello <strong>world</strong></p>"
	html, err := RichTextToHTML(src)
	if err != nil {
		t.Fatal(err)
	}
	if html != src {
		t.Fatalf("html = %q", html)
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
