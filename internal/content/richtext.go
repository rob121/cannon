package content

import (
	"strings"

	"github.com/rob121/cannon/internal/markdown"
)

// RichTextToHTML renders stored item intro/body for the frontend.
// HTML from the admin editor is passed through; legacy Markdown is converted.
func RichTextToHTML(src string) (string, error) {
	if IsHTMLContent(src) {
		return src, nil
	}
	return markdown.ToHTML(src)
}

// IsHTMLContent reports whether src looks like HTML from the WYSIWYG editor.
func IsHTMLContent(src string) bool {
	s := strings.TrimSpace(src)
	if s == "" {
		return false
	}
	if strings.HasPrefix(s, "<") {
		return true
	}
	return false
}
