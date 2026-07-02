package templateengine

import (
	"strings"
	"testing"

	"github.com/rob121/cannon/internal/config"
)

func testAdminFuncs() map[string]any {
	return map[string]any{
		"listQuery":       func(page int, sort, dir string) string { return "?" },
		"listQueryAmp":    func(page int, sort, dir string) string { return "" },
		"sortLink":        func(basePath string, page int, currentSort, currentDir, col string) string { return basePath },
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
	}
}

func TestAdminTemplateLookup(t *testing.T) {
	e := New(&config.SiteConfig{}, nil, nil, testAdminFuncs())
	for _, name := range []string{
		"admin/layout.html",
		"admin/dashboard.html",
		"admin/users.html",
	} {
		if _, err := e.parse(name); err != nil {
			t.Fatalf("parse %s: %v", name, err)
		}
	}
}

func TestPaginationWithInt64Total(t *testing.T) {
	e := New(&config.SiteConfig{}, nil, nil, testAdminFuncs())
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
