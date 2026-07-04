package admin

import (
	"html/template"
	"net/http"
	"net/url"
	"strings"

	"github.com/rob121/cannon/internal/extensions"
	"github.com/rob121/cannon/internal/help"
	"github.com/rob121/cannon/internal/markdown"
	"github.com/rob121/cannon/internal/sites"
)

const helpBase = "/admin/help"

type helpCrumb struct {
	Label   string
	URL     string
	Current bool
}

type helpBrowseItem struct {
	Label string
	URL   string
	Count int
	Kind  string
}

func (h *Handler) help(w http.ResponseWriter, r *http.Request, path string) {
	site, _ := sites.FromContext(r.Context())
	extMgr := h.chain.Extensions(site)
	_ = extMgr.Bootstrap(r.Context())

	internalSections, err := help.Index()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	extensionSections, err := extMgr.HelpIndex(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	parts := pathParts("/help", path)
	title := "Help"
	data := map[string]any{
		"ActiveNav":         "help",
		"Subtitle":          "Browse documentation folders and guides.",
		"InternalSections":  internalSections,
		"ExtensionSections": extensionSections,
		"BasePath":          helpBase,
	}

	switch {
	case len(parts) == 0:
		data["Breadcrumbs"] = []helpCrumb{{Label: "All Help", URL: helpBase, Current: true}}
		data["Folders"] = buildRootFolders(internalSections, extensionSections)
	case parts[0] == "extensions":
		h.helpExtensionsPath(w, r, parts, internalSections, extensionSections, extMgr, data, &title)
		return
	default:
		h.helpInternalPath(w, r, parts, internalSections, data, &title)
		return
	}

	h.render(w, r, title, "admin/help.html", data)
}

func (h *Handler) helpInternalPath(w http.ResponseWriter, r *http.Request, parts []string, sections []help.Section, data map[string]any, title *string) {
	folder := parts[0]
	section, ok := help.FindSection(sections, folder)
	if !ok {
		h.notFound(w, r)
		return
	}

	if len(parts) == 1 {
		crumbs := []helpCrumb{
			{Label: "All Help", URL: helpBase},
			{Label: section.Label, URL: help.FolderURL(folder), Current: true},
		}
		data["Breadcrumbs"] = crumbs
		data["Folders"] = nil
		data["Files"] = internalFileItems(*section)
		h.render(w, r, section.Label, "admin/help.html", data)
		return
	}

	slug := parts[1]
	article, ok := help.FindArticle(sections, folder, slug)
	if !ok {
		h.notFound(w, r)
		return
	}
	md, err := help.Fetch(folder, slug)
	if err != nil {
		h.notFound(w, r)
		return
	}
	html, err := markdown.ToHTML(md)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	*title = article.Title
	data["Breadcrumbs"] = []helpCrumb{
		{Label: "All Help", URL: helpBase},
		{Label: section.Label, URL: help.FolderURL(folder)},
		{Label: article.Title, URL: help.ArticleURL(folder, slug), Current: true},
	}
	data["ContentHTML"] = template.HTML(html)
	h.render(w, r, *title, "admin/help.html", data)
}

func (h *Handler) helpExtensionsPath(w http.ResponseWriter, r *http.Request, parts []string, internalSections []help.Section, sections []extensions.HelpSection, extMgr *extensions.Manager, data map[string]any, title *string) {
	if len(parts) == 1 {
		data["Breadcrumbs"] = []helpCrumb{
			{Label: "All Help", URL: helpBase},
			{Label: "Extensions", URL: help.ExtensionsRootURL(), Current: true},
		}
		data["ExtensionFolders"] = extensionFolderItems(sections)
		if sec, ok := help.FindSection(internalSections, "extensions"); ok {
			data["BuiltinGuides"] = builtinExtensionFileItems(*sec)
		}
		h.render(w, r, "Extensions", "admin/help.html", data)
		return
	}

	extName, _ := url.PathUnescape(parts[1])
	if extName == "_" {
		h.helpBuiltinExtensionDocs(w, r, parts, internalSections, data, title)
		return
	}
	section, ok := findExtensionSection(sections, extName)
	if !ok {
		h.notFound(w, r)
		return
	}

	if len(parts) == 2 {
		if len(section.Articles) > 0 {
			redirectList(w, r, extensions.HelpArticleURL(extName, section.Articles[0].Path))
			return
		}
		h.renderExtensionHelpReader(w, r, extName, section, "", extMgr, data, title)
		return
	}

	articlePath := extensions.HelpArticlePathFromParts(parts[2:])
	h.renderExtensionHelpReader(w, r, extName, section, articlePath, extMgr, data, title)
}

func (h *Handler) renderExtensionHelpReader(w http.ResponseWriter, r *http.Request, extName string, section *extensions.HelpSection, articlePath string, extMgr *extensions.Manager, data map[string]any, title *string) {
	data["ViewMode"] = "extension-reader"
	data["ExtensionLabel"] = section.Label
	data["ExtensionName"] = extName
	data["ExtensionArticles"] = section.Articles

	crumbs := []helpCrumb{
		{Label: "All Help", URL: helpBase},
		{Label: "Extensions", URL: help.ExtensionsRootURL()},
		{Label: section.Label, URL: extensions.ExtensionFolderURL(extName)},
	}

	if articlePath != "" {
		article, ok := findHelpArticle([]extensions.HelpSection{*section}, extName, articlePath)
		if !ok {
			h.notFound(w, r)
			return
		}
		md, err := extMgr.FetchHelpArticle(extName, articlePath)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}
		html, err := markdown.ToHTML(md)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		*title = article.Title
		data["ActivePath"] = articlePath
		data["ContentHTML"] = template.HTML(html)
		crumbs = append(crumbs, helpCrumb{
			Label:   article.Title,
			URL:     extensions.HelpArticleURL(extName, articlePath),
			Current: true,
		})
	} else {
		*title = section.Label
		crumbs[len(crumbs)-1].Current = true
	}

	data["Breadcrumbs"] = crumbs
	h.render(w, r, *title, "admin/help.html", data)
}

func buildRootFolders(internal []help.Section, ext []extensions.HelpSection) []helpBrowseItem {
	out := make([]helpBrowseItem, 0, len(internal)+1)
	if sec, ok := help.FindSection(internal, "admin"); ok {
		out = append(out, internalFolderItem(*sec))
	}
	extDocCount := 0
	if sec, ok := help.FindSection(internal, "extensions"); ok {
		extDocCount = len(sec.Articles)
	}
	if len(ext) > 0 || extDocCount > 0 {
		count := len(ext)
		if count == 0 {
			count = extDocCount
		}
		out = append(out, helpBrowseItem{
			Label: "Extension Docs",
			URL:   help.ExtensionsRootURL(),
			Count: count,
			Kind:  "folder",
		})
	}
	if sec, ok := help.FindSection(internal, "getting-started"); ok {
		out = append(out, internalFolderItem(*sec))
	}
	for _, sec := range internal {
		if sec.Folder == "admin" || sec.Folder == "getting-started" || sec.Folder == "extensions" {
			continue
		}
		out = append(out, internalFolderItem(sec))
	}
	return out
}

func internalFolderItem(sec help.Section) helpBrowseItem {
	return helpBrowseItem{
		Label: sec.Label,
		URL:   help.FolderURL(sec.Folder),
		Count: len(sec.Articles),
		Kind:  "folder",
	}
}

func internalFileItems(sec help.Section) []helpBrowseItem {
	out := make([]helpBrowseItem, 0, len(sec.Articles))
	for _, article := range sec.Articles {
		out = append(out, helpBrowseItem{
			Label: article.Title,
			URL:   help.ArticleURL(article.Folder, article.Slug),
			Kind:  "file",
		})
	}
	return out
}

func builtinExtensionFileItems(sec help.Section) []helpBrowseItem {
	out := make([]helpBrowseItem, 0, len(sec.Articles))
	for _, article := range sec.Articles {
		out = append(out, helpBrowseItem{
			Label: article.Title,
			URL:   help.BuiltinExtensionArticleURL(article.Slug),
			Kind:  "file",
		})
	}
	return out
}

func (h *Handler) helpBuiltinExtensionDocs(w http.ResponseWriter, r *http.Request, parts []string, internalSections []help.Section, data map[string]any, title *string) {
	sec, ok := help.FindSection(internalSections, "extensions")
	if !ok {
		h.notFound(w, r)
		return
	}
	if len(parts) == 2 {
		if len(sec.Articles) > 0 {
			redirectList(w, r, help.BuiltinExtensionArticleURL(sec.Articles[0].Slug))
			return
		}
		h.notFound(w, r)
		return
	}
	slug := parts[2]
	article, ok := help.FindArticle(internalSections, "extensions", slug)
	if !ok {
		h.notFound(w, r)
		return
	}
	md, err := help.Fetch("extensions", slug)
	if err != nil {
		h.notFound(w, r)
		return
	}
	html, err := markdown.ToHTML(md)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	*title = article.Title
	data["Breadcrumbs"] = []helpCrumb{
		{Label: "All Help", URL: helpBase},
		{Label: "Extensions", URL: help.ExtensionsRootURL()},
		{Label: article.Title, URL: help.BuiltinExtensionArticleURL(slug), Current: true},
	}
	data["ContentHTML"] = template.HTML(html)
	h.render(w, r, *title, "admin/help.html", data)
}

func extensionFolderItems(sections []extensions.HelpSection) []helpBrowseItem {
	out := make([]helpBrowseItem, 0, len(sections))
	for _, sec := range sections {
		out = append(out, helpBrowseItem{
			Label: sec.Label,
			URL:   extensions.ExtensionFolderURL(sec.Extension),
			Count: len(sec.Articles),
			Kind:  "folder",
		})
	}
	return out
}

func extensionFileItems(sec extensions.HelpSection) []helpBrowseItem {
	out := make([]helpBrowseItem, 0, len(sec.Articles))
	for _, article := range sec.Articles {
		out = append(out, helpBrowseItem{
			Label: article.Title,
			URL:   extensions.HelpArticleURL(article.Extension, article.Path),
			Kind:  "file",
		})
	}
	return out
}

func findExtensionSection(sections []extensions.HelpSection, extensionName string) (*extensions.HelpSection, bool) {
	for i := range sections {
		if sections[i].Extension == extensionName {
			return &sections[i], true
		}
	}
	return nil, false
}

func findHelpArticle(sections []extensions.HelpSection, extensionName, articlePath string) (*extensions.HelpArticle, bool) {
	articlePath = normalizeHelpArticlePath(articlePath)
	for _, sec := range sections {
		if sec.Extension != extensionName {
			continue
		}
		for _, article := range sec.Articles {
			if normalizeHelpArticlePath(article.Path) == articlePath {
				return &article, true
			}
		}
	}
	return nil, false
}

func normalizeHelpArticlePath(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return "/"
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	return path
}
