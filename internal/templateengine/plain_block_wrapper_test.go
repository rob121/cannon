package templateengine

import (
	"html/template"
	"strings"
	"testing"

	"github.com/rob121/cannon/internal/themes"
)

func TestPlainBlockWrapperTemplate(t *testing.T) {
	e := New(nil, themes.Selection{}, nil, nil, testFrontendFuncs())
	tmpl, err := e.parse("default/partials/blocks/plain.html")
	if err != nil {
		t.Fatal(err)
	}
	var buf strings.Builder
	if err := tmpl.Execute(&buf, map[string]any{"Body": template.HTML("<p>Block body</p>")}); err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(buf.String()) != "<p>Block body</p>" {
		t.Fatalf("got %q", buf.String())
	}
}
