package blocks_test

import (
	"bytes"
	"context"
	"html/template"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/rob121/cannon/internal/blocks"
	"github.com/rob121/cannon/internal/config"
	"github.com/rob121/cannon/internal/database"
	"github.com/rob121/cannon/internal/lang"
	"github.com/rob121/cannon/internal/models"
	"github.com/rob121/cannon/internal/settings"
	"github.com/rob121/cannon/internal/sites"
	"github.com/rob121/cannon/internal/templateengine"
	"github.com/rob121/cannon/internal/themes"
)

func testAdminFuncs() template.FuncMap {
	funcs := template.FuncMap{
		"listQuery":          func(page int, sort, dir string) string { return "?" },
		"listQueryFromData":  func(root map[string]any) string { return "" },
		"pageSizeLinkRoot":   func(root map[string]any, size int) string { return "/admin/test?page_size=25" },
		"sortLink":           func(basePath string, page int, currentSort, currentDir, col string) string { return basePath },
		"sortLinkRoot":       func(root map[string]any, col string) string { return "/admin/test" },
		"containsUint":       func(ids []uint, id uint) bool { return false },
		"containsString":     func(list []string, value string) bool { return false },
		"providerName":       func(name string) string { return name },
		"uintPtrEq":          func(a *uint, b uint) bool { return a != nil && *a == b },
		"groupName":          func(name string) string { return name },
		"roleName":           func(name string) string { return name },
		"joinRoleNames":      func(roles any) string { return "" },
		"joinGroupNames":     func(groups any) string { return "" },
		"helpURL":            func(extensionName, articlePath string) string { return "#" },
		"internalHelpURL":    func(folder, slug string) string { return "#" },
		"siteURL":            func(host string) string { return "#" },
		"siteAdminURL":       func(host string) string { return "#" },
		"siteHostLabel":      func(host string) string { return host },
		"csrfField":          func() template.HTML { return "" },
		"csrfToken":          func() string { return "test-csrf" },
		"showRouteTitle":     func() bool { return true },
		"fieldOptions":       func(string) []map[string]string { return nil },
		"fieldValueContains": func(string, string) bool { return false },
		"formatBytes":        func(int64) string { return "0 B" },
		"initials":           func(parts ...string) string { return "?" },
	}
	for k, v := range lang.TestFuncMap() {
		funcs[k] = v
	}
	return funcs
}

func TestAdminDashboardSpaceRowsRender(t *testing.T) {
	templateengine.InvalidateAll()
	site := &config.SiteConfig{
		ID:       "admin-dash-space-debug",
		Name:     "Dash",
		Database: config.DatabaseConfig{Type: "sqlite", DSN: t.TempDir() + "/site.db"},
	}
	if err := database.Migrate(site); err != nil {
		t.Fatal(err)
	}
	ctx := sites.WithContext(context.Background(), site)
	db, err := database.Get(site.ID)
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&models.Group{Name: "public", Status: models.StatusActive}).Error; err != nil {
		t.Fatal(err)
	}
	if err := settings.Save(ctx, settings.NewStore(), settings.ScopeGlobal, settings.SectionGeneral, map[string]any{
		"debug_template_spaces": true,
		"admin_theme":           "admin",
	}); err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodGet, "/admin?tp=1", nil).WithContext(ctx)
	debug := blocks.DebugSpacesActive(req.Context(), req)
	if !debug {
		t.Fatal("expected DebugSpacesActive")
	}
	var eng *templateengine.Engine
	blockRenderer := func(name string) (template.HTML, error) {
		var fragment blocks.FragmentRenderer
		if eng != nil {
			fragment = func(tmpl string, data map[string]any) (string, error) {
				var buf bytes.Buffer
				if err := eng.RenderFragment(&buf, tmpl, data); err != nil {
					return "", err
				}
				return buf.String(), nil
			}
		}
		html, err := blocks.RenderSpace(req.Context(), nil, name, req, map[string]any{"authenticated": true}, fragment)
		if err != nil {
			return "", err
		}
		return template.HTML(html), nil
	}
	blockLen := func(name string) (int, error) {
		n, err := blocks.CountForSpace(req.Context(), nil, name)
		return blocks.PreviewCount(n, err, name, debug)
	}
	eng = templateengine.New(site, themes.Selection{Admin: themes.BuiltinAdmin}, blockRenderer, blockLen, testAdminFuncs())
	if err := blocks.EnsureAdminDashboardBlocks(req.Context()); err != nil {
		t.Fatal(err)
	}
	tmpl, err := eng.ParseNested("admin/dashboard.html")
	if err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, map[string]any{
		"Title":     "Dashboard",
		"ActiveNav": "dashboard",
		"Subtitle":  "x",
	}); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	for _, want := range []string{
		`data-space="admin-dashboard-top"`,
		`data-space="admin-dashboard-widgets"`,
		`data-space="admin-dashboard-main"`,
		"admin-dashboard-row",
		"Review Queue",
		"Quick Start",
		"Update Check",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q (len=%d)", want, len(out))
		}
	}
}
