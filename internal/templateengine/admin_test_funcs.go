package templateengine

import (
	"html/template"

	"github.com/rob121/cannon/internal/lang"
	"github.com/rob121/cannon/internal/models"
)

// TestAdminFuncs returns stub template funcs for parsing embedded admin templates in tests.
func TestAdminFuncs() template.FuncMap {
	funcs := template.FuncMap{
		"listQuery":            func(page int, sort, dir string) string { return "?" },
		"listQueryFromData":      func(root map[string]any) string { return "" },
		"pageSizeLinkRoot":       func(root map[string]any, size int) string { return "/admin/test?page_size=25" },
		"sortLink":               func(basePath string, page int, currentSort, currentDir, col string) string { return basePath },
		"sortLinkRoot":           func(root map[string]any, col string) string { return "/admin/test" },
		"containsUint":           func(ids []uint, id uint) bool { return false },
		"containsString":         func(list []string, value string) bool { return false },
		"providerName":           func(name string) string { return name },
		"uintPtrEq":              func(a *uint, b uint) bool { return a != nil && *a == b },
		"groupName":              func(name string) string { return name },
		"roleName":               func(name string) string { return name },
		"joinRoleNames":          func(roles any) string { return "" },
		"joinGroupNames":         func(groups any) string { return "" },
		"helpURL":                func(extensionName, articlePath string) string { return "#" },
		"internalHelpURL":        func(folder, slug string) string { return "#" },
		"siteURL":                func(host string) string { return "#" },
		"siteAdminURL":           func(host string) string { return "#" },
		"siteHostLabel":          func(host string) string { return host },
		"csrfField":              func() template.HTML { return "" },
		"csrfToken":              func() string { return "test-csrf" },
		"showRouteTitle":         func() bool { return true },
		"fieldOptions":           func(string) []map[string]string { return nil },
		"fieldValueContains":     func(string, string) bool { return false },
		"formatBytes":            func(int64) string { return "0 B" },
		"initials":               func(parts ...string) string { return "?" },
		"blockMetaSelected":      func(map[string]any, string, string) bool { return false },
		"blockMetaTruthy":        func(map[string]any, string) bool { return false },
		"blockTypeOptionLabel":   func(t models.BlockType) string { return string(t) },
	}
	for k, v := range lang.TestFuncMap() {
		funcs[k] = v
	}
	return funcs
}
