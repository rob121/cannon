package admin

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/rob121/cannon/internal/config"
	"github.com/rob121/cannon/internal/sites"
	"github.com/rob121/cannon/internal/templatemeta"
	"github.com/rob121/cannon/internal/templatemgr"
	"github.com/rob121/cannon/internal/themes"
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
	case len(parts) == 1 && parts[0] == "import":
		h.templateImport(w, r, site)
	case len(parts) == 1 && parts[0] == "edit":
		h.templateEdit(w, r, site)
	case len(parts) == 1 && parts[0] == "override":
		h.templateOverride(w, r, site)
	case len(parts) == 1 && parts[0] == "revert":
		h.templateRevert(w, r, site)
	case len(parts) == 2 && parts[1] == "meta":
		h.templateMeta(w, r, site, parts[0])
	case len(parts) == 2 && parts[1] == "assets":
		h.templateAssets(w, r, site, parts[0])
	case len(parts) == 3 && parts[1] == "assets" && parts[2] == "edit":
		h.templateAssetEdit(w, r, site, parts[0])
	case len(parts) == 3 && parts[1] == "assets" && parts[2] == "new":
		h.templateAssetNew(w, r, site, parts[0])
	case len(parts) == 3 && parts[1] == "assets" && parts[2] == "delete":
		h.templateAssetDelete(w, r, site, parts[0])
	case len(parts) == 1:
		h.templateGroup(w, r, site, parts[0])
	default:
		h.notFound(w, r)
	}
}

func (h *Handler) templateRoot(w http.ResponseWriter, r *http.Request, site *config.SiteConfig) {
	groups, err := templatemgr.ListThemes(site.TemplateDir)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	data := listPage(r, 1, int64(len(groups)), templatesBase,
		"Browse installable themes. Each theme is a self-contained folder with HTML templates, assets, and template.json metadata.",
		"New Theme", map[string]any{
			"ActiveNav":                "templates",
			"TemplatesBase":            templatesBase,
			"PageActionURL":            templatesBase + "/new?create=theme",
			"PageActionLabel":          "New Theme",
			"PageSecondaryActionURL":   templatesBase + "/import",
			"PageSecondaryActionLabel": "Import from Git",
		})
	col, dir, _ := parseSort(r, map[string]string{
		"name": "name", "type": "type", "status": "status", "total": "total",
	}, "name")
	data["Sort"] = col
	data["Dir"] = dir
	templatemgr.SortGroups(convertThemeSummaries(groups), col, dir)
	data["Groups"] = convertThemeSummaries(groups)
	data["TemplatesBase"] = templatesBase
	if errMsg := strings.TrimSpace(r.URL.Query().Get("error")); errMsg != "" {
		data["Error"] = errMsg
	}
	h.render(w, r, "Templates", "admin/templates.html", data)
}

func convertThemeSummaries(themesList []themes.Summary) []templatemgr.GroupSummary {
	out := make([]templatemgr.GroupSummary, 0, len(themesList))
	for _, theme := range themesList {
		out = append(out, templatemgr.GroupSummary{
			Name:   theme.Name,
			Label:  theme.Label,
			Type:   theme.Type,
			Status: theme.Status,
			Total:  theme.Total,
		})
	}
	return out
}

func (h *Handler) templateGroup(w http.ResponseWriter, r *http.Request, site *config.SiteConfig, group string) {
	if err := templatemgr.ValidateTheme(group); err != nil {
		h.notFound(w, r)
		return
	}
	entries, err := templatemgr.ThemeTemplates(site.TemplateDir, group)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	meta, err := templatemeta.Load(themes.Root(site.TemplateDir, group))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	overridden := 0
	for _, entry := range entries {
		if entry.Overridden {
			overridden++
		}
	}
	data := listPage(r, 1, int64(len(entries)), templatesBase,
		fmt.Sprintf("Override or edit templates in the %s theme.", group),
		"New File", map[string]any{
			"ActiveNav":       "templates",
			"Group":           group,
			"Overridden":      overridden,
			"Breadcrumbs":     templateBreadcrumbs(group, ""),
			"PageActionURL":   templatesBase + "/new?group=" + url.QueryEscape(group),
			"PageActionLabel": "New File",
			"TemplatesBase":   templatesBase,
			"AssetsURL":       templatesBase + "/" + url.PathEscape(group) + "/assets",
			"MetaURL":         templatesBase + "/" + url.PathEscape(group) + "/meta",
		})
	if errMsg := strings.TrimSpace(r.URL.Query().Get("error")); errMsg != "" {
		data["Error"] = errMsg
	}
	col, dir, _ := parseSort(r, map[string]string{
		"name": "name", "source": "source", "status": "status",
	}, "name")
	data["Sort"] = col
	data["Dir"] = dir
	rootEntries, folders := templatemgr.GroupEntriesByFolder(entries, col, dir)
	data["RootEntries"] = rootEntries
	data["Folders"] = folders
	data["TemplateTotal"] = len(entries)
	data["FolderTotal"] = len(folders)
	data["BasePath"] = templatesBase + "/" + group
	data["TemplatesBase"] = templatesBase
	data["GroupLabel"] = meta.Name
	if data["GroupLabel"] == "" {
		data["GroupLabel"] = group
	}
	data["GroupType"] = meta.Type
	data["GroupTypeLabel"] = themeTypeLabel(meta.Type)
	data["GroupStatus"] = meta.Status
	h.render(w, r, group+" Theme", "admin/templates_group.html", data)
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
		h.notFound(w, r)
		return
	}
	group := templatemgr.ThemeFromPath(path)
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

func (h *Handler) templateRevert(w http.ResponseWriter, r *http.Request, site *config.SiteConfig) {
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
		h.notFound(w, r)
		return
	}
	group := templatemgr.ThemeFromPath(path)
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

	if err := templatemgr.RevertOverride(site.TemplateDir, path); err != nil {
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
		action := formString(r, "action")
		if action == "create_theme" {
			if err := h.createTheme(site, r); err != nil {
				h.renderThemeCreateForm(w, r, site, err.Error())
				return
			}
			redirectList(w, r, templatesBase)
			return
		}
		path := formString(r, "path")
		content := r.FormValue("content")
		if err := templatemgr.Save(site.TemplateDir, path, []byte(content)); err != nil {
			h.renderTemplateForm(w, r, site, path, content, false, "", templatemgr.ThemeFromPath(path), err.Error(), templatemgr.CanRevertOverride(site.TemplateDir, path))
			return
		}
		redirectList(w, r, templatesBase+"/edit?path="+url.QueryEscape(path))
		return
	}

	if r.URL.Query().Get("create") == "theme" {
		h.renderThemeCreateForm(w, r, site, "")
		return
	}

	path := templatemgr.CleanRelPath(r.URL.Query().Get("path"))
	group := r.URL.Query().Get("group")
	content := ""
	fromBuiltin := false
	source := ""
	if path != "" {
		var err error
		content, source, fromBuiltin, err = templatemgr.Read(site.TemplateDir, path)
		if err != nil {
			h.notFound(w, r)
			return
		}
	}
	h.renderTemplateForm(w, r, site, path, content, fromBuiltin, source, group, "", templatemgr.CanRevertOverride(site.TemplateDir, path))
}

func (h *Handler) createTheme(site *config.SiteConfig, r *http.Request) error {
	name := slugify(formString(r, "theme_name"))
	if err := themes.ValidateName(name); err != nil {
		return err
	}
	themeRoot := themes.Root(site.TemplateDir, name)
	if _, err := os.Stat(themeRoot); err == nil {
		return fmt.Errorf("theme %q already exists", name)
	}
	if err := os.MkdirAll(filepath.Join(themeRoot, "assets"), 0755); err != nil {
		return err
	}
	meta := templatemeta.DefaultPackMeta()
	meta.Name = strings.TrimSpace(formString(r, "theme_label"))
	if meta.Name == "" {
		meta.Name = name
	}
	meta.Type = formString(r, "theme_type")
	if meta.Type == "" {
		meta.Type = templatemeta.TypeFull
	}
	meta.Author = formString(r, "theme_author")
	meta.Description = formString(r, "theme_description")
	return templatemeta.Save(themeRoot, meta)
}

func (h *Handler) templateImport(w http.ResponseWriter, r *http.Request, site *config.SiteConfig) {
	switch r.Method {
	case http.MethodGet:
		h.renderTemplateImportForm(w, r, site, templateImportForm{}, "")
	case http.MethodPost:
		if err := r.ParseForm(); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		form := templateImportForm{
			RepoURL:          formString(r, "repo_url"),
			Branch:           formString(r, "branch"),
			ThemeName:        formString(r, "theme_name"),
			ThemeLabel:       formString(r, "theme_label"),
			ThemeType:        formString(r, "theme_type"),
			ThemeAuthor:      formString(r, "theme_author"),
			ThemeDescription: formString(r, "theme_description"),
		}
		switch formString(r, "action") {
		case "fetch_branches":
			repoURL, err := templatemgr.NormalizeGitURL(form.RepoURL)
			if err != nil {
				h.renderTemplateImportForm(w, r, site, form, err.Error())
				return
			}
			form.RepoURL = repoURL
			if strings.TrimSpace(form.ThemeName) == "" {
				form.ThemeName = templatemgr.ThemeNameFromRepoURL(repoURL)
			}
			branches, err := templatemgr.ListGitBranches(r.Context(), repoURL)
			if err != nil {
				h.renderTemplateImportForm(w, r, site, form, err.Error())
				return
			}
			form.Branches = branches
			if form.Branch == "" {
				form.Branch = defaultGitBranch(branches)
			}
			if form.ThemeType == "" {
				form.ThemeType = templatemeta.TypeFull
			}
			h.renderTemplateImportForm(w, r, site, form, "")
		case "import_theme":
			if strings.TrimSpace(form.ThemeName) == "" {
				form.ThemeName = templatemgr.ThemeNameFromRepoURL(form.RepoURL)
			}
			if form.ThemeType == "" {
				form.ThemeType = templatemeta.TypeFull
			}
			err := templatemgr.ImportGitTheme(r.Context(), templatemgr.GitImportOptions{
				TemplateDir: site.TemplateDir,
				RepoURL:     form.RepoURL,
				Branch:      form.Branch,
				ThemeName:   form.ThemeName,
				Label:       form.ThemeLabel,
				Type:        form.ThemeType,
				Author:      form.ThemeAuthor,
				Description: form.ThemeDescription,
			})
			if err != nil {
				branches, listErr := templatemgr.ListGitBranches(context.Background(), form.RepoURL)
				if listErr == nil {
					form.Branches = branches
				}
				h.renderTemplateImportForm(w, r, site, form, err.Error())
				return
			}
			themeName := slugify(form.ThemeName)
			if themeName == "" {
				themeName = templatemgr.ThemeNameFromRepoURL(form.RepoURL)
			}
			redirectList(w, r, templatesBase+"/"+url.PathEscape(themeName))
		default:
			http.Error(w, "unknown action", http.StatusBadRequest)
		}
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

type templateImportForm struct {
	RepoURL          string
	Branch           string
	ThemeName        string
	ThemeLabel       string
	ThemeType        string
	ThemeAuthor      string
	ThemeDescription string
	Branches         []string
}

func defaultGitBranch(branches []string) string {
	for _, preferred := range []string{"main", "master", "develop"} {
		for _, branch := range branches {
			if branch == preferred {
				return branch
			}
		}
	}
	if len(branches) > 0 {
		return branches[0]
	}
	return ""
}

func (h *Handler) renderTemplateImportForm(w http.ResponseWriter, r *http.Request, site *config.SiteConfig, form templateImportForm, errMsg string) {
	data := formData(map[string]any{
		"ActiveNav":        "templates",
		"TemplatesBase":    templatesBase,
		"RepoURL":          form.RepoURL,
		"Branch":           form.Branch,
		"ThemeName":        form.ThemeName,
		"ThemeLabel":       form.ThemeLabel,
		"ThemeType":        form.ThemeType,
		"ThemeAuthor":      form.ThemeAuthor,
		"ThemeDescription": form.ThemeDescription,
		"Branches":         form.Branches,
	})
	if errMsg != "" {
		data["Error"] = errMsg
	}
	h.render(w, r, "Import Theme from Git", "admin/templates_import.html", data)
}

func (h *Handler) renderThemeCreateForm(w http.ResponseWriter, r *http.Request, site *config.SiteConfig, errMsg string) {
	data := formData(map[string]any{
		"ActiveNav":     "templates",
		"TemplatesBase": templatesBase,
		"CancelURL":     templatesBase,
		"IsThemeCreate": true,
	})
	if errMsg != "" {
		data["Error"] = errMsg
	}
	h.render(w, r, "New Theme", "admin/templates_form.html", data)
}

func (h *Handler) templateEdit(w http.ResponseWriter, r *http.Request, site *config.SiteConfig) {
	if r.Method == http.MethodPost {
		if err := r.ParseForm(); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		path := templatemgr.CleanRelPath(formString(r, "path"))
		if path == "" {
			h.notFound(w, r)
			return
		}
		content := r.FormValue("content")
		if err := templatemgr.Save(site.TemplateDir, path, []byte(content)); err != nil {
			h.renderTemplateForm(w, r, site, path, content, false, "", templatemgr.ThemeFromPath(path), err.Error(), templatemgr.CanRevertOverride(site.TemplateDir, path))
			return
		}
		redirectList(w, r, templatesBase+"/edit?path="+url.QueryEscape(path))
		return
	}

	path := templatemgr.CleanRelPath(r.URL.Query().Get("path"))
	if path == "" {
		h.notFound(w, r)
		return
	}

	content, source, fromBuiltin, err := templatemgr.Read(site.TemplateDir, path)
	if err != nil {
		h.notFound(w, r)
		return
	}
	canRevert := templatemgr.CanRevertOverride(site.TemplateDir, path)
	h.renderTemplateForm(w, r, site, path, content, fromBuiltin, source, templatemgr.ThemeFromPath(path), "", canRevert)
}

func (h *Handler) renderTemplateForm(w http.ResponseWriter, r *http.Request, site *config.SiteConfig, path, content string, fromBuiltin bool, source, group, errMsg string, canRevert bool) {
	creating := r.URL.Path == templatesBase+"/new" && r.Method != http.MethodPost
	title := "Edit Template"
	if creating && path == "" {
		title = "New File"
	} else if creating {
		title = "Override Template"
	}
	if group == "" {
		group = templatemgr.ThemeFromPath(path)
	}
	cancelURL := templatesBase
	if group != "" {
		cancelURL = templatesBase + "/" + group
	}
	data := formData(map[string]any{
		"ActiveNav":     "templates",
		"Path":          path,
		"Content":       content,
		"FromBuiltin":   fromBuiltin,
		"Source":        source,
		"TemplateDir":   site.TemplateDir,
		"BasePath":      templatesBase,
		"TemplatesBase": templatesBase,
		"CancelURL":     cancelURL,
		"Group":         group,
		"IsNew":         creating && path == "",
		"IsCreate":      creating,
		"Breadcrumbs":   templateBreadcrumbs(group, path),
		"CanRevert":     canRevert,
	})
	if errMsg != "" {
		data["Error"] = errMsg
	}
	h.render(w, r, title, "admin/templates_form.html", data)
}

func (h *Handler) templateMeta(w http.ResponseWriter, r *http.Request, site *config.SiteConfig, theme string) {
	if err := templatemgr.ValidateTheme(theme); err != nil {
		h.notFound(w, r)
		return
	}
	themeRoot := themes.Root(site.TemplateDir, theme)

	if r.Method == http.MethodPost {
		if err := r.ParseForm(); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		meta := templatemeta.PackMeta{
			Name:        formString(r, "name"),
			Type:        formString(r, "type"),
			Author:      formString(r, "author"),
			Description: formString(r, "description"),
			Version:     formString(r, "version"),
			Status:      formString(r, "status"),
		}
		if err := templatemeta.Save(themeRoot, meta); err != nil {
			h.renderTemplateMetaForm(w, r, site, theme, meta, err.Error())
			return
		}
		redirectList(w, r, templatesBase+"/"+url.PathEscape(theme))
		return
	}

	meta, err := templatemeta.Load(themeRoot)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	h.renderTemplateMetaForm(w, r, site, theme, meta, "")
}

func (h *Handler) renderTemplateMetaForm(w http.ResponseWriter, r *http.Request, site *config.SiteConfig, theme string, meta templatemeta.PackMeta, errMsg string) {
	data := formData(map[string]any{
		"ActiveNav":     "templates",
		"Meta":          meta,
		"Theme":         theme,
		"BasePath":      templatesBase + "/" + url.PathEscape(theme) + "/meta",
		"TemplatesBase": templatesBase,
		"CancelURL":     templatesBase + "/" + url.PathEscape(theme),
		"Breadcrumbs": []map[string]string{
			{"Label": "Templates", "URL": templatesBase},
			{"Label": theme, "URL": templatesBase + "/" + url.PathEscape(theme)},
			{"Label": "Theme Metadata", "URL": ""},
		},
	})
	if errMsg != "" {
		data["Error"] = errMsg
	}
	h.render(w, r, "Theme Metadata", "admin/templates_meta.html", data)
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
