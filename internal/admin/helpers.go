package admin

import (
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/rob121/cannon/internal/config"
	"github.com/rob121/cannon/internal/groups"
	"github.com/rob121/cannon/internal/httpx"
	"github.com/rob121/cannon/internal/models"
	"github.com/rob121/cannon/internal/sites"
	"github.com/rob121/cannon/internal/settings"
	"github.com/rob121/cannon/internal/user"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
	"gorm.io/gorm"
)

// ActivateFunc reloads server routes after config changes.
type ActivateFunc func(*config.App) error

// ReloadFunc reloads configuration from disk and reactivates the server.
type ReloadFunc func() error

func pathParts(prefix, path string) []string {
	rest := strings.TrimPrefix(path, prefix)
	rest = strings.TrimPrefix(rest, "/")
	if rest == "" {
		return nil
	}
	return strings.Split(rest, "/")
}

func parseID(s string) (uint, bool) {
	n, err := strconv.ParseUint(s, 10, 64)
	if err != nil || n == 0 {
		return 0, false
	}
	return uint(n), true
}

func formString(r *http.Request, key string) string {
	return strings.TrimSpace(r.FormValue(key))
}

func themeTypeLabel(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "frontend":
		return "Frontend"
	case "backend":
		return "Admin"
	case "full":
		return "Full"
	case "":
		return "Full"
	default:
		return strings.ToUpper(raw[:1]) + strings.ToLower(raw[1:])
	}
}

func formStatus(r *http.Request) models.Status {
	if formString(r, "status") == "inactive" {
		return models.StatusInactive
	}
	return models.StatusActive
}

func formInt(r *http.Request, key string, fallback int) int {
	v, err := strconv.Atoi(strings.TrimSpace(r.FormValue(key)))
	if err != nil {
		return fallback
	}
	return v
}

func formBool(r *http.Request, key string) bool {
	v := strings.TrimSpace(r.FormValue(key))
	return v == "on" || v == "1" || v == "true" || v == "yes"
}

func formUintPtr(r *http.Request, key string) *uint {
	s := strings.TrimSpace(r.FormValue(key))
	if s == "" || s == "0" {
		return nil
	}
	n, err := strconv.ParseUint(s, 10, 64)
	if err != nil || n == 0 {
		return nil
	}
	v := uint(n)
	return &v
}

func formItemStatus(r *http.Request) models.ItemStatus {
	switch formString(r, "status") {
	case string(models.ItemStatusPublished):
		return models.ItemStatusPublished
	case string(models.ItemStatusPending):
		return models.ItemStatusPending
	case string(models.ItemStatusArchived):
		return models.ItemStatusArchived
	case string(models.ItemStatusTrashed):
		return models.ItemStatusTrashed
	default:
		return models.ItemStatusDraft
	}
}

func formTimePtr(r *http.Request, key string) *time.Time {
	s := strings.TrimSpace(r.FormValue(key))
	if s == "" {
		return nil
	}
	for _, layout := range []string{"2006-01-02T15:04", "2006-01-02 15:04", "2006-01-02"} {
		if t, err := time.Parse(layout, s); err == nil {
			return &t
		}
	}
	return nil
}

func redirectList(w http.ResponseWriter, r *http.Request, base string) {
	httpx.RedirectSeeOther(w, r, base)
}

func slugify(v string) string {
	v = strings.ToLower(strings.TrimSpace(v))
	re := regexp.MustCompile(`[^a-z0-9]+`)
	v = re.ReplaceAllString(v, "-")
	v = strings.Trim(v, "-")
	if v == "" {
		return "site"
	}
	return v
}

const defaultPageSize = 25

func pageSizeFor(r *http.Request) int {
	if r == nil {
		return defaultPageSize
	}
	limit, err := settings.DefaultListLimit(r.Context())
	if err != nil || limit <= 0 {
		return defaultPageSize
	}
	return limit
}

func listData(page int, total int64, size int, extra map[string]any) map[string]any {
	data := map[string]any{
		"Page": page, "Total": total, "PageSize": size,
		"Sort": "", "Dir": "asc",
	}
	for k, v := range extra {
		data[k] = v
	}
	return data
}

func listPage(r *http.Request, page int, total int64, basePath, subtitle, addLabel string, extra map[string]any) map[string]any {
	data := listData(page, total, pageSizeFor(r), extra)
	if basePath != "" {
		data["BasePath"] = basePath
	}
	if subtitle != "" {
		data["Subtitle"] = subtitle
	}
	if addLabel != "" {
		data["PageActionURL"] = basePath + "/new"
		data["PageActionLabel"] = addLabel
	}
	return data
}

func formData(extra map[string]any) map[string]any {
	data := map[string]any{"IsForm": true}
	for k, v := range extra {
		data[k] = v
	}
	return data
}

func layoutContext(r *http.Request) map[string]any {
	data := map[string]any{}
	if site, err := sites.FromContext(r.Context()); err == nil {
		data["SiteName"] = site.Name
		data["SiteURL"] = siteFrontendURL(site.Host)
		data["SiteHostLabel"] = siteHostLabel(site.Host)
	}
	if svc, err := user.FromContext(r.Context()); err == nil {
		if u, err := svc.Current(r.Context()); err == nil {
			displayName := strings.TrimSpace(strings.TrimSpace(u.GivenName + " " + u.FamilyName))
			if displayName == "" {
				displayName = u.Username
			}
			data["CurrentUser"] = map[string]any{
				"ID":          u.UserID,
				"Username":    u.Username,
				"GivenName":   u.GivenName,
				"FamilyName":  u.FamilyName,
				"DisplayName": displayName,
			}
		}
	}
	navCan := navCanMap(r.Context())
	data["NavCan"] = navCan
	data["NavContentVisible"] = navGroupVisible(navCan, "items", "review", "trash", "categories", "tags", "field_groups", "media", "comments")
	data["NavMenusVisible"] = navGroupVisible(navCan, "menus", "menu_items")
	data["NavUsersVisible"] = navGroupVisible(navCan, "accounts", "authenticators", "profiles", "groups", "roles", "permissions")
	data["NavAPIVisible"] = navGroupVisible(navCan, "api_credentials", "api_settings")
	data["NavSystemVisible"] = navGroupVisible(navCan, "sites", "extension_registry", "extension_apps", "blocks", "configuration", "notifications", "access_log")
	return data
}

func dashboardWelcomeSubtitle(r *http.Request) string {
	if svc, err := user.FromContext(r.Context()); err == nil {
		if u, err := svc.Current(r.Context()); err == nil {
			name := strings.TrimSpace(strings.TrimSpace(u.GivenName + " " + u.FamilyName))
			if name == "" {
				name = u.Username
			}
			if name != "" {
				return "Welcome back, " + name + ". Here's an overview of your site today."
			}
		}
	}
	return "Welcome back. Here's an overview of your site today."
}

func siteFrontendURL(host string) string {
	host = strings.TrimSpace(host)
	if host == "" {
		return "/"
	}
	if !strings.Contains(host, "://") {
		return "http://" + host
	}
	return host
}

func siteHostLabel(host string) string {
	host = strings.TrimSpace(host)
	if host == "" {
		return "Site"
	}
	if !strings.Contains(host, "://") {
		host = "http://" + host
	}
	u, err := url.Parse(host)
	if err != nil || u.Host == "" {
		return host
	}
	return u.Host
}

func siteAdminURL(host string) string {
	return strings.TrimRight(siteFrontendURL(host), "/") + "/admin"
}

func absoluteSiteURL(host, path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return siteFrontendURL(host)
	}
	if strings.HasPrefix(path, "http://") || strings.HasPrefix(path, "https://") {
		return path
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	return strings.TrimRight(siteFrontendURL(host), "/") + path
}

func formUintList(r *http.Request, key string) []uint {
	var ids []uint
	for _, s := range r.Form[key] {
		if id, ok := parseID(s); ok {
			ids = append(ids, id)
		}
	}
	return ids
}

func containsUint(list []uint, id uint) bool {
	for _, v := range list {
		if v == id {
			return true
		}
	}
	return false
}

func uintPtrEq(ptr *uint, id uint) bool {
	return ptr != nil && *ptr == id
}

func usersNavOpen(nav string) bool {
	switch nav {
	case "accounts", "authenticators", "profiles", "groups", "roles", "permissions":
		return true
	default:
		return false
	}
}

func menusNavOpen(nav string) bool {
	switch nav {
	case "menus", "menu_items":
		return true
	default:
		return false
	}
}

func contentNavOpen(nav string) bool {
	switch nav {
	case "items", "categories", "tags", "field_groups", "comments", "media", "trash", "review":
		return true
	default:
		return false
	}
}

func systemNavOpen(nav string) bool {
	switch nav {
	case "sites", "extension_registry", "extension_apps", "blocks", "configuration", "notifications", "access_log":
		return true
	default:
		return false
	}
}

func extensionAppsNavOpen(nav string) bool {
	return strings.HasPrefix(nav, "extension_app:")
}

type AdminExtensionNav struct {
	Name      string
	URL       string
	ActiveKey string
}

func formatBytes(n int64) string {
	switch {
	case n >= 1024*1024:
		return fmt.Sprintf("%.1f MiB", float64(n)/(1024*1024))
	case n >= 1024:
		return fmt.Sprintf("%.1f KiB", float64(n)/1024)
	default:
		return fmt.Sprintf("%d B", n)
	}
}

func loadActiveProfiles(db *gorm.DB) []models.Profile {
	var rows []models.Profile
	db.Order("name asc").Find(&rows)
	return rows
}

func loadActiveGroups(db *gorm.DB) []models.Group {
	var rows []models.Group
	db.Where("status = ?", models.StatusActive).Order("name asc").Find(&rows)
	return rows
}

// loadMembershipGroups returns backend organizational groups for user/group RBAC assignment.
func loadMembershipGroups(db *gorm.DB) []models.Group {
	var rows []models.Group
	db.Where("status = ? AND kind = ?", models.StatusActive, models.GroupKindBackend).Order("name asc").Find(&rows)
	return rows
}

// loadFrontendGroups returns visibility groups for content, routes, and blocks.
func loadFrontendGroups(db *gorm.DB) []models.Group {
	var rows []models.Group
	db.Where("status = ? AND kind = ?", models.StatusActive, models.GroupKindFrontend).Order("name asc").Find(&rows)
	return rows
}

func groupSelectedIDs(groups []models.Group) []uint {
	selected := make([]uint, 0, len(groups))
	for _, group := range groups {
		selected = append(selected, group.GroupID)
	}
	return selected
}

func defaultGroupSelectedIDs(db *gorm.DB, assigned []models.Group, isNew bool) []uint {
	selected := groupSelectedIDs(assigned)
	if len(selected) > 0 || !isNew {
		return selected
	}
	if id, err := groups.PublicGroupID(db); err == nil {
		return []uint{id}
	}
	return selected
}

func replaceFormGroups(db *gorm.DB, model any, r *http.Request) error {
	groupIDs := formUintList(r, "group_ids")
	if len(groupIDs) == 0 {
		return fmt.Errorf("select at least one group")
	}
	var selected []models.Group
	if err := db.Where("group_id IN ?", groupIDs).Find(&selected).Error; err != nil {
		return err
	}
	if len(selected) == 0 {
		return fmt.Errorf("select at least one group")
	}
	return db.Model(model).Association("Groups").Replace(selected)
}

func replaceFormGroupsOptional(db *gorm.DB, model any, assocName string, r *http.Request, formKey string) error {
	groupIDs := formUintList(r, formKey)
	if len(groupIDs) == 0 {
		return db.Model(model).Association(assocName).Clear()
	}
	var selected []models.Group
	if err := db.Where("group_id IN ?", groupIDs).Find(&selected).Error; err != nil {
		return err
	}
	return db.Model(model).Association(assocName).Replace(selected)
}

func isProtectedGroupName(name string) bool {
	switch name {
	case groups.PublicGroupName, groups.RegisteredGroupName,
		groups.AdministratorsGroupName, groups.ManagerGroupName,
		groups.EditorGroupName, groups.WriterGroupName:
		return true
	default:
		return false
	}
}

// GroupDisplayName returns the admin-facing label for a group name.
func GroupDisplayName(name string) string {
	switch name {
	case groups.PublicGroupName:
		return "Public"
	case groups.RegisteredGroupName:
		return "Registered"
	default:
		return name
	}
}

// RoleDisplayName returns the admin-facing label for a role name.
func RoleDisplayName(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return ""
	}
	name = strings.ReplaceAll(name, "_", " ")
	name = strings.ReplaceAll(name, "-", " ")
	return cases.Title(language.English, cases.NoLower).String(name)
}
