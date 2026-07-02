package admin

import (
	"html/template"
	"net/http"
	"net/url"
	"strings"

	"github.com/rob121/cannon/internal/models"
	"github.com/rob121/cannon/internal/sites"
	"github.com/rob121/cannon/internal/user"
	"gorm.io/gorm"
)

const extensionAppsBase = "/admin/extension-apps"

func (h *Handler) extensionApps(w http.ResponseWriter, r *http.Request, path string) {
	parts := pathParts("/extension-apps", path)
	if len(parts) == 0 {
		redirectList(w, r, extensionsBase)
		return
	}
	name, err := url.PathUnescape(parts[0])
	if err != nil || name == "" {
		h.notFound(w, r)
		return
	}
	suffix := ""
	if len(parts) > 1 {
		suffix = "/" + strings.Join(parts[1:], "/")
	}
	h.extensionAppProxy(w, r, name, suffix)
}

func (h *Handler) extensionAppProxy(w http.ResponseWriter, r *http.Request, name, suffixPath string) {
	site, _ := sites.FromContext(r.Context())
	extMgr := h.chain.Extensions(site)
	_ = extMgr.Bootstrap(r.Context())

	rt, ok := extMgr.Runtime(name)
	if !ok || rt.Capabilities.Admin == "" {
		h.notFound(w, r)
		return
	}
	userCtx, err := extensionAdminUserContext(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	html, err := extMgr.RenderAdmin(r.Context(), name, suffixPath, r, userCtx)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	db, _ := sites.DB(r.Context())
	title := extensionMenuLabel(name, extensionMenuNames(db))
	h.render(w, r, title, "admin/extension_app.html", map[string]any{
		"ActiveNav": "extension_app:" + name,
		"Subtitle":  "Extension admin interface.",
		"Extension": name,
		"Content":   template.HTML(html),
	})
}

func extensionAdminUserContext(r *http.Request) (map[string]any, error) {
	svc, err := user.FromContext(r.Context())
	if err != nil {
		return map[string]any{"authenticated": false}, nil
	}
	return svc.Context(r.Context())
}

func (h *Handler) adminExtensionNav(r *http.Request) []AdminExtensionNav {
	site, err := sites.FromContext(r.Context())
	if err != nil {
		return nil
	}
	db, _ := sites.DB(r.Context())
	menuNames := extensionMenuNames(db)
	items := make([]AdminExtensionNav, 0)
	for _, rt := range h.chain.Extensions(site).AdminRuntimes() {
		items = append(items, AdminExtensionNav{
			Name:      extensionMenuLabel(rt.Model.Name, menuNames),
			URL:       extensionAppsBase + "/" + url.PathEscape(rt.Model.Name),
			ActiveKey: "extension_app:" + rt.Model.Name,
		})
	}
	return items
}

func extensionMenuNames(db *gorm.DB) map[string]string {
	names := map[string]string{}
	if db == nil {
		return names
	}
	var rows []models.Extension
	db.Select("name", "menu_name").Find(&rows)
	for _, row := range rows {
		names[row.Name] = row.MenuName
	}
	return names
}

func extensionMenuLabel(binaryName string, menuNames map[string]string) string {
	if label := menuNames[binaryName]; label != "" {
		return label
	}
	return binaryName
}
