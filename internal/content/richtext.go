package content

import (
	"context"
	"strings"

	"github.com/rob121/cannon/internal/content/htmlpolicy"
	"github.com/rob121/cannon/internal/markdown"
)

// RichTextToHTML renders stored item intro/body for the frontend.
// HTML from the admin editor is sanitized; legacy Markdown is converted then sanitized.
func RichTextToHTML(ctx context.Context, src string) (string, error) {
	policy, hosts := htmlPolicyForContext(ctx)
	if IsHTMLContent(src) {
		return htmlpolicy.SanitizeContentHTML(policy, hosts, src), nil
	}
	html, err := markdown.ToHTML(src)
	if err != nil {
		return "", err
	}
	return htmlpolicy.SanitizeContentHTML(policy, hosts, html), nil
}

// SanitizeStoredRichText normalizes HTML before persistence; Markdown is stored unchanged.
func SanitizeStoredRichText(ctx context.Context, src string) string {
	src = strings.TrimSpace(src)
	if src == "" {
		return ""
	}
	if !IsHTMLContent(src) {
		return src
	}
	policy, hosts := htmlPolicyForContext(ctx)
	return htmlpolicy.SanitizeContentHTML(policy, hosts, src)
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
