package admin

import (
	"fmt"
	"net/http"
	"net/url"
	"path/filepath"
	"strings"

	"github.com/rob121/cannon/internal/config"
	"github.com/rob121/cannon/internal/templatemgr"
	"github.com/rob121/cannon/internal/templatemeta"
	"github.com/rob121/cannon/internal/themes"
)

func (h *Handler) templateAssets(w http.ResponseWriter, r *http.Request, site *config.SiteConfig, theme string) {
	if err := templatemgr.ValidateTheme(theme); err != nil {
		h.notFound(w, r)
		return
	}
	entries, err := templatemgr.ListThemeAssets(site.TemplateDir, theme)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	meta, err := templatemeta.Load(themes.Root(site.TemplateDir, theme))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	editable := 0
	for _, entry := range entries {
		if entry.Editable {
			editable++
		}
	}
	basePath := templatesBase + "/" + url.PathEscape(theme) + "/assets"
	data := listPage(r, 1, int64(len(entries)), basePath,
		fmt.Sprintf("CSS, JavaScript, and other static files served from /theme/ for the %s theme.", theme),
		"New Asset", map[string]any{
			"ActiveNav":       "templates",
			"Theme":           theme,
			"Breadcrumbs":     assetBreadcrumbs(theme, ""),
			"PageActionURL":   basePath + "/new",
			"PageActionLabel": "New Asset",
			"TemplatesBase":   templatesBase,
			"ThemeURL":        templatesBase + "/" + url.PathEscape(theme),
		})
	if errMsg := strings.TrimSpace(r.URL.Query().Get("error")); errMsg != "" {
		data["Error"] = errMsg
	}
	col, dir, _ := parseSort(r, map[string]string{
		"name": "name", "type": "type", "editable": "editable",
	}, "name")
	data["Sort"] = col
	data["Dir"] = dir
	rootEntries, folders := templatemgr.GroupAssetsByFolder(entries, col, dir)
	data["RootEntries"] = rootEntries
	data["Folders"] = folders
	data["AssetTotal"] = len(entries)
	data["EditableTotal"] = editable
	data["FolderTotal"] = len(folders)
	data["BasePath"] = basePath
	data["GroupLabel"] = meta.Name
	if data["GroupLabel"] == "" {
		data["GroupLabel"] = theme
	}
	h.render(w, r, theme+" Assets", "admin/templates_assets.html", data)
}

func (h *Handler) templateAssetEdit(w http.ResponseWriter, r *http.Request, site *config.SiteConfig, theme string) {
	if err := templatemgr.ValidateTheme(theme); err != nil {
		h.notFound(w, r)
		return
	}
	if r.Method == http.MethodPost {
		if err := r.ParseForm(); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		path := templatemgr.CleanRelPath(formString(r, "path"))
		if path == "" || templatemgr.ThemeFromAssetPath(path) != theme {
			h.notFound(w, r)
			return
		}
		content := r.FormValue("content")
		if err := templatemgr.SaveAsset(site.TemplateDir, path, []byte(content)); err != nil {
			h.renderAssetForm(w, r, site, theme, path, content, true, err.Error())
			return
		}
		redirectList(w, r, templatesBase+"/"+url.PathEscape(theme)+"/assets/edit?path="+url.QueryEscape(path))
		return
	}

	path := templatemgr.CleanRelPath(r.URL.Query().Get("path"))
	if path == "" || templatemgr.ThemeFromAssetPath(path) != theme {
		h.notFound(w, r)
		return
	}
	content, err := templatemgr.ReadAsset(site.TemplateDir, path)
	if err != nil {
		h.notFound(w, r)
		return
	}
	h.renderAssetForm(w, r, site, theme, path, content, true, "")
}

func (h *Handler) templateAssetNew(w http.ResponseWriter, r *http.Request, site *config.SiteConfig, theme string) {
	if err := templatemgr.ValidateTheme(theme); err != nil {
		h.notFound(w, r)
		return
	}
	if r.Method == http.MethodPost {
		if err := r.ParseForm(); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		relative := strings.TrimPrefix(filepath.ToSlash(strings.TrimSpace(formString(r, "relative"))), "/")
		path := templatemgr.AssetPath(theme, relative)
		content := r.FormValue("content")
		if err := templatemgr.SaveAsset(site.TemplateDir, path, []byte(content)); err != nil {
			h.renderAssetForm(w, r, site, theme, path, content, false, err.Error())
			return
		}
		redirectList(w, r, templatesBase+"/"+url.PathEscape(theme)+"/assets/edit?path="+url.QueryEscape(path))
		return
	}

	path := templatemgr.CleanRelPath(r.URL.Query().Get("path"))
	if path != "" && templatemgr.ThemeFromAssetPath(path) == theme {
		content, err := templatemgr.ReadAsset(site.TemplateDir, path)
		if err != nil {
			h.notFound(w, r)
			return
		}
		h.renderAssetForm(w, r, site, theme, path, content, true, "")
		return
	}
	h.renderAssetForm(w, r, site, theme, "", "", false, "")
}

func (h *Handler) templateAssetDelete(w http.ResponseWriter, r *http.Request, site *config.SiteConfig, theme string) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if err := templatemgr.ValidateTheme(theme); err != nil {
		h.notFound(w, r)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	path := templatemgr.CleanRelPath(formString(r, "path"))
	if path == "" {
		path = templatemgr.CleanRelPath(r.URL.Query().Get("path"))
	}
	if path == "" || templatemgr.ThemeFromAssetPath(path) != theme {
		h.notFound(w, r)
		return
	}
	basePath := templatesBase + "/" + url.PathEscape(theme) + "/assets"
	if err := templatemgr.DeleteAsset(site.TemplateDir, path); err != nil {
		redirectList(w, r, appendQueryParam(basePath, "error", err.Error()))
		return
	}
	redirectList(w, r, basePath)
}

func (h *Handler) renderAssetForm(w http.ResponseWriter, r *http.Request, site *config.SiteConfig, theme, path, content string, editing bool, errMsg string) {
	title := "New Asset"
	if editing {
		title = "Edit Asset"
	}
	basePath := templatesBase + "/" + url.PathEscape(theme) + "/assets"
	relative := templatemgr.AssetRelative(path)
	ext := strings.ToLower(strings.TrimPrefix(filepath.Ext(path), "."))
	data := formData(map[string]any{
		"ActiveNav":     "templates",
		"Theme":         theme,
		"Path":          path,
		"Relative":      relative,
		"Content":       content,
		"EditorMode":    templatemgr.EditorModeForAsset(ext),
		"BasePath":      basePath,
		"TemplatesBase": templatesBase,
		"CancelURL":     basePath,
		"ThemeURL":      templatesBase + "/" + url.PathEscape(theme),
		"IsEdit":        editing,
		"CanDelete":     editing && path != "",
		"Breadcrumbs":   assetBreadcrumbs(theme, relative),
	})
	if errMsg != "" {
		data["Error"] = errMsg
	}
	h.render(w, r, title, "admin/templates_asset_form.html", data)
}

func assetBreadcrumbs(theme, relative string) []map[string]string {
	crumbs := []map[string]string{
		{"Label": "Templates", "URL": templatesBase},
		{"Label": theme, "URL": templatesBase + "/" + url.PathEscape(theme)},
		{"Label": "Assets", "URL": templatesBase + "/" + url.PathEscape(theme) + "/assets"},
	}
	if relative != "" {
		crumbs = append(crumbs, map[string]string{
			"Label": relative,
			"URL":   "",
		})
	}
	return crumbs
}
