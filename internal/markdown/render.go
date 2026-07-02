package markdown

import (
	"bytes"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer/html"
)

var converter = goldmark.New(
	goldmark.WithExtensions(extension.GFM),
	goldmark.WithParserOptions(parser.WithAutoHeadingID()),
	goldmark.WithRendererOptions(html.WithHardWraps()),
)

// ToHTML converts markdown source to HTML.
func ToHTML(src string) (string, error) {
	var buf bytes.Buffer
	if err := converter.Convert([]byte(src), &buf); err != nil {
		return "", err
	}
	return buf.String(), nil
}
