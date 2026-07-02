package templateengine

import (
	"strings"
	"testing"

	"github.com/rob121/cannon/internal/config"
	"github.com/rob121/cannon/internal/themes"
)

func testAdminFuncs() map[string]any {
	return map[string]any{
		"listQuery":       func(page int, sort, dir string) string { return "?" },
		"listQueryFromData": func(root map[string]any) string { return "" },
		"sortLink":        func(basePath string, page int, currentSort, currentDir, col string) string { return basePath },
		"sortLinkRoot":    func(root map[string]any, col string) string { return "/admin/test" },
		"containsUint":    func(ids []uint, id uint) bool { return false },
		"uintPtrEq":       func(a *uint, b uint) bool { return a != nil && *a == b },
		"groupName":       func(name string) string { return name },
		"joinRoleNames":   func(roles any) string { return "" },
		"joinGroupNames":  func(groups any) string { return "" },
		"helpURL":         func(extensionName, articlePath string) string { return "#" },
		"internalHelpURL": func(folder, slug string) string { return "#" },
		"siteURL":         func(host string) string { return "#" },
		"siteAdminURL":    func(host string) string { return "#" },
		"siteHostLabel":   func(host string) string { return host },
		"csrfField":       func() string { return "" },
		"csrfToken":       func() string { return "test-csrf" },
		"showRouteTitle":  func() bool { return true },
	}
}

func TestAdminTemplateLookup(t *testing.T) {
	e := New(&config.SiteConfig{}, themes.Selection{}, nil, nil, testAdminFuncs())
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
	e := New(&config.SiteConfig{}, themes.Selection{}, nil, nil, testAdminFuncs())
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
	e := New(&config.SiteConfig{}, themes.Selection{}, nil, nil, testAdminFuncs())
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
