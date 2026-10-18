package server

import (
	"bytes"
	"errors"
	"html/template"
	"strings"
	"testing"

	"github.com/rob121/cannon/internal/blocks"
	"github.com/rob121/cannon/internal/templateengine"
)

func TestConditionalSpacePreview(t *testing.T) {
	for _, enabled := range []bool{false, true} {
		for _, condition := range []string{`gt (lenspace "sidebar") 0`, `hasSpace "sidebar"`} {
			t.Run(condition+"/preview="+map[bool]string{false: "off", true: "on"}[enabled], func(t *testing.T) {
				count := func(name string) (int, error) {
					return blocks.PreviewCount(0, nil, name, enabled)
				}
				render := func(name string) (template.HTML, error) {
					if enabled {
						return template.HTML(blocks.WrapDebugSpace(name, "")), nil
					}
					return "", nil
				}
				tmpl, err := template.New("layout").Funcs(templateengine.FuncMap(render, count)).Parse(
					`{{if ` + condition + `}}<aside>{{space "sidebar"}}</aside>{{else}}fallback{{end}}`)
				if err != nil {
					t.Fatal(err)
				}
				var out bytes.Buffer
				if err := tmpl.Execute(&out, nil); err != nil {
					t.Fatal(err)
				}
				if !enabled {
					if out.String() != "fallback" {
						t.Fatalf("normal rendering changed: %s", out.String())
					}
					return
				}
				for _, want := range []string{"<aside>", "border:2px dashed #FF3300", ">sidebar</div></div></aside>"} {
					if !strings.Contains(out.String(), want) {
						t.Fatalf("preview missing %q: %s", want, out.String())
					}
				}
				if strings.Contains(out.String(), "fallback") {
					t.Fatalf("preview used fallback instead of outline: %s", out.String())
				}
			})
		}
	}
}

func TestSpacePreviewPreservesCountsAndErrors(t *testing.T) {
	lookupErr := errors.New("block lookup failed")
	for _, tc := range []struct {
		name  string
		count int
		err   error
	}{
		{name: "sidebar", count: 2},
		{name: "sidebar", err: lookupErr},
		{name: ""},
		{name: "  "},
	} {
		got, err := blocks.PreviewCount(tc.count, tc.err, tc.name, true)
		if got != tc.count || !errors.Is(err, tc.err) {
			t.Fatalf("PreviewCount(%q) = %d, %v; want %d, %v", tc.name, got, err, tc.count, tc.err)
		}
	}
}
