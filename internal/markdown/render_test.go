package markdown

import (
	"strings"
	"testing"
)

func TestToHTMLRendersGFMTables(t *testing.T) {
	src := `| Name | Value |
|------|:-----:|
| one  | 1     |
`
	html, err := ToHTML(src)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(html, "<table>") {
		t.Fatalf("expected table HTML, got %q", html)
	}
	if !strings.Contains(html, "<th") || !strings.Contains(html, "<td") {
		t.Fatalf("expected table cells, got %q", html)
	}
}

func TestToHTMLRendersHeadingsAndCode(t *testing.T) {
	src := "# Title\n\n`code`\n"
	html, err := ToHTML(src)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(html, "<h1") || !strings.Contains(html, "<code>code</code>") {
		t.Fatalf("unexpected html: %q", html)
	}
}
