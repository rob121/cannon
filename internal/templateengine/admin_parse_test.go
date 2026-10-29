package templateengine

import (
	"strings"
	"testing"

	"github.com/rob121/cannon/internal/config"
	"github.com/rob121/cannon/internal/models"
	"github.com/rob121/cannon/internal/themes"
)

func TestCategoryRelatedItemsControl(t *testing.T) {
	InvalidateAll()
	e := New(&config.SiteConfig{ID: t.Name()}, themes.Selection{}, nil, nil, TestAdminFuncs())
	tmpl, err := e.parse("admin/categories_form.html")
	if err != nil {
		t.Fatal(err)
	}
	for _, mode := range []string{"", "inherit", "yes", "no"} {
		var out strings.Builder
		if err := tmpl.Execute(&out, map[string]any{
			"Row":            models.Category{ItemRelatedDisplay: mode},
			"ContentLocales": []string{"en-US"},
		}); err != nil {
			t.Fatal(err)
		}
		_, control, found := strings.Cut(out.String(), `name="item_related_display">`)
		if !found {
			t.Fatal("missing Show Related Items control")
		}
		control, _, _ = strings.Cut(control, "</select>")
		want := mode
		if want == "" {
			want = "inherit"
		}
		if !strings.Contains(control, `value="`+want+`" selected`) || strings.Count(control, "selected") != 1 {
			t.Fatalf("mode %q has incorrect selected option: %s", mode, control)
		}
	}
}

func TestAdminTemplateLookup(t *testing.T) {
	InvalidateAll()
	e := New(&config.SiteConfig{ID: t.Name()}, themes.Selection{}, nil, nil, TestAdminFuncs())
	for _, name := range []string{
		"admin/layout.html",
		"admin/dashboard.html",
		"admin/users.html",
		"admin/menu_items.html",
		"admin/blocks_form.html",
	} {
		if _, err := e.parse(name); err != nil {
			t.Fatalf("parse %s: %v", name, err)
		}
	}
}

func TestSortHeaderRendered(t *testing.T) {
	e := New(&config.SiteConfig{}, themes.Selection{}, nil, nil, TestAdminFuncs())
	for _, name := range []string{"admin/blocks.html", "admin/categories.html", "admin/comments.html", "admin/menu_items.html"} {
		tmpl, err := e.parse(name)
		if err != nil {
			t.Fatalf("parse %s: %v", name, err)
		}
		var buf strings.Builder
		err = tmpl.Execute(&buf, map[string]any{
			"Rows":     nil,
			"Page":     1,
			"Total":    int64(0),
			"PageSize": 25,
			"BasePath": "/admin/test",
			"Sort":     "name",
			"Dir":      "asc",
		})
		if err != nil {
			t.Fatalf("execute %s: %v", name, err)
		}
		out := buf.String()
		if !strings.Contains(out, "admin-sort-icons") {
			t.Fatalf("%s: missing sort header markup", name)
		}
		if !strings.Contains(out, "bi-chevron-up") {
			t.Fatalf("%s: missing sort chevrons", name)
		}
	}
}

func TestPaginationWithInt64Total(t *testing.T) {
	e := New(&config.SiteConfig{}, themes.Selection{}, nil, nil, TestAdminFuncs())
	tmpl, err := e.parse("admin/users.html")
	if err != nil {
		t.Fatalf("parse users.html: %v", err)
	}
	var buf strings.Builder
	err = tmpl.Execute(&buf, map[string]any{
		"Rows":     nil,
		"Page":     1,
		"Total":    int64(25),
		"PageSize": 20,
		"BasePath": "/admin/users",
		"Sort":     "name",
		"Dir":      "asc",
	})
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
}
