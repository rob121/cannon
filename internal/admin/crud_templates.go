package admin

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/rob121/cannon/internal/config"
	"github.com/rob121/cannon/internal/sites"
	"github.com/rob121/cannon/internal/templatemgr"
)

const templatesBase = "/admin/templates"

func (h *Handler) templates(w http.ResponseWriter, r *http.Request, path string) {
	site, err := sites.FromContext(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	parts := pathParts("/templates", path)
	switch {
	case len(parts) == 0:
		h.templateRoot(w, r, site)
	case len(parts) == 1 && parts[0] == "new":
		h.templateNew(w, r, site)
	case len(parts) == 1 && parts[0] == "edit":
		h.templateEdit(w, r, site)
	case len(parts) == 1 && parts[0] == "override":
		h.templateOverride(w, r, site)
	case len(parts) == 1:
		h.templateGroup(w, r, site, parts[0])
	default:
		http.NotFound(w, r)
	}
}

func (h *Handler) templateRoot(w http.ResponseWriter, r *http.Request, site *config.SiteConfig) {
	groups, err := templatemgr.RootGroups(site.TemplateDir)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	data := listPage(1, int64(len(groups)), templatesBase,
		"Browse built-in template groups and manage site overrides.",
		"", map[string]any{"ActiveNav": "templates", "TemplatesBase": templatesBase})
	col, dir, _ := parseSort(r, map[string]string{
		"name": "name", "total": "total", "overridden": "overridden",
	}, "name")
	data["Sort"] = col
	data["Dir"] = dir
	templatemgr.SortGroups(groups, col, dir)
	data["Groups"] = groups
	data["TemplatesBase"] = templatesBase
	if errMsg := strings.TrimSpace(r.URL.Query().Get("error")); errMsg != "" {
		data["Error"] = errMsg
	}
	h.render(w, r, "Templates", "admin/templates.html", data)
}

func (h *Handler) templateGroup(w http.ResponseWriter, r *http.Request, site *config.SiteConfig, group string) {
	if err := templatemgr.ValidateGroup(group); err != nil {
		http.NotFound(w, r)
		return
	}
	entries, err := templatemgr.GroupTemplates(site.TemplateDir, group)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if len(entries) == 0 {
		http.NotFound(w, r)
		return
	}
	overridden := 0
	for _, entry := range entries {
		if entry.Overridden {
			overridden++
		}
	}
	data := listPage(1, int64(len(entries)), templatesBase,
		fmt.Sprintf("Override or edit %s templates. Custom files replace built-in versions.", group),
		"New Template", map[string]any{
			"ActiveNav":       "templates",
			"Group":           group,
			"Overridden":      overridden,
			"Breadcrumbs":     templateBreadcrumbs(group, ""),
			"PageActionURL":   templatesBase + "/new?group=" + url.QueryEscape(group),
			"PageActionLabel": "New Template",
			"TemplatesBase":   templatesBase,
		})
	if errMsg := strings.TrimSpace(r.URL.Query().Get("error")); errMsg != "" {
		data["Error"] = errMsg
	}
	col, dir, _ := parseSort(r, map[string]string{
		"name": "name", "source": "source", "status": "status",
	}, "name")
	data["Sort"] = col
	data["Dir"] = dir
	templatemgr.SortEntries(entries, col, dir)
	data["Entries"] = entries
	data["BasePath"] = templatesBase + "/" + group
	data["TemplatesBase"] = templatesBase
	h.render(w, r, group+" Templates", "admin/templates_group.html", data)
}

func (h *Handler) templateOverride(w http.ResponseWriter, r *http.Request, site *config.SiteConfig) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	path := templatemgr.CleanRelPath(r.FormValue("path"))
	if path == "" {
		path = templatemgr.CleanRelPath(r.URL.Query().Get("path"))
	}
	if path == "" {
		http.NotFound(w, r)
		return
	}
	group := templatemgr.GroupFromPath(path)
	returnURL := templatesBase
	if group != "" {
		returnURL = templatesBase + "/" + group
	}
	sort := strings.TrimSpace(r.FormValue("sort"))
	if sort == "" {
		sort = r.URL.Query().Get("sort")
	}
	dir := strings.TrimSpace(r.FormValue("dir"))
	if dir == "" {
		dir = r.URL.Query().Get("dir")
	}
	if sort != "" {
		returnURL += listQuery(1, sort, dir)
	}

	if err := templatemgr.Override(site.TemplateDir, path); err != nil {
		returnURL = appendQueryParam(returnURL, "error", err.Error())
	}
	redirectList(w, r, returnURL)
}

func appendQueryParam(rawURL, key, value string) string {
	if strings.TrimSpace(value) == "" {
		return rawURL
	}
	sep := "?"
	if strings.Contains(rawURL, "?") {
		sep = "&"
	}
	return rawURL + sep + url.QueryEscape(key) + "=" + url.QueryEscape(value)
}

func (h *Handler) templateNew(w http.ResponseWriter, r *http.Request, site *config.SiteConfig) {
	if r.Method == http.MethodPost {
		if err := r.ParseForm(); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		path := formString(r, "path")
		content := r.FormValue("content")
		if err := templatemgr.Save(site.TemplateDir, path, []byte(content)); err != nil {
			h.renderTemplateForm(w, r, site, path, content, false, "", templatemgr.GroupFromPath(path), err.Error())
			return
		}
		redirectList(w, r, templatesBase+"/edit?path="+url.QueryEscape(path))
		return
	}

	path := r.URL.Query().Get("path")
	group := r.URL.Query().Get("group")
	content := ""
	fromBuiltin := false
	source := ""
	if path != "" {
		var err error
		content, source, fromBuiltin, err = templatemgr.Read(site.TemplateDir, path)
		if err != nil {
			http.NotFound(w, r)
			return
		}
	}
	h.renderTemplateForm(w, r, site, path, content, fromBuiltin, source, group, "")
}

func (h *Handler) templateEdit(w http.ResponseWriter, r *http.Request, site *config.SiteConfig) {
	path := r.URL.Query().Get("path")
	if path == "" {
		http.NotFound(w, r)
		return
	}

	if r.Method == http.MethodPost {
		if err := r.ParseForm(); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		path = formString(r, "path")
		content := r.FormValue("content")
		if err := templatemgr.Save(site.TemplateDir, path, []byte(content)); err != nil {
			h.renderTemplateForm(w, r, site, path, content, false, "", templatemgr.GroupFromPath(path), err.Error())
			return
		}
		redirectList(w, r, templatesBase+"/edit?path="+url.QueryEscape(path))
		return
	}

	content, source, fromBuiltin, err := templatemgr.Read(site.TemplateDir, path)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	h.renderTemplateForm(w, r, site, path, content, fromBuiltin, source, templatemgr.GroupFromPath(path), "")
}

func (h *Handler) renderTemplateForm(w http.ResponseWriter, r *http.Request, site *config.SiteConfig, path, content string, fromBuiltin bool, source, group, errMsg string) {
	creating := r.URL.Path == templatesBase+"/new" && r.Method != http.MethodPost
	title := "Edit Template"
	if creating && path == "" {
		title = "New Template"
	} else if creating {
		title = "Override Template"
	}
	if group == "" {
		group = templatemgr.GroupFromPath(path)
	}
	cancelURL := templatesBase
	if group != "" {
		cancelURL = templatesBase + "/" + group
	}
	data := formData(map[string]any{
		"ActiveNav":   "templates",
		"Path":        path,
		"Content":     content,
		"FromBuiltin": fromBuiltin,
		"Source":      source,
		"TemplateDir": site.TemplateDir,
		"BasePath":    templatesBase,
		"TemplatesBase": templatesBase,
		"CancelURL":   cancelURL,
		"Group":       group,
		"IsNew":       creating && path == "",
		"IsCreate":    creating,
		"Breadcrumbs": templateBreadcrumbs(group, path),
	})
	if errMsg != "" {
		data["Error"] = errMsg
	}
	h.render(w, r, title, "admin/templates_form.html", data)
}

func templateBreadcrumbs(group, path string) []map[string]string {
	crumbs := []map[string]string{
		{"Label": "Templates", "URL": templatesBase},
	}
	if group == "" {
		return crumbs
	}
	crumbs = append(crumbs, map[string]string{
		"Label": group,
		"URL":   templatesBase + "/" + group,
	})
	if path != "" {
		crumbs = append(crumbs, map[string]string{
			"Label": path[strings.LastIndex(path, "/")+1:],
			"URL":   "",
		})
	}
	return crumbs
}
