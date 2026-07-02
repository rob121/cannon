package admin

import (
	"net/http"
	"sort"
	"strings"

	"github.com/rob121/cannon/internal/lang"
	"github.com/rob121/cannon/internal/sites"
)

const languagesBase = "/admin/languages"

func (h *Handler) languages(w http.ResponseWriter, r *http.Request, path string) {
	parts := pathParts("/languages", path)
	switch {
	case len(parts) == 0:
		h.languageList(w, r)
	case len(parts) == 1:
		h.languageFile(w, r, parts[0])
	default:
		h.notFound(w, r)
	}
}

func (h *Handler) langMgr(r *http.Request) (*lang.Manager, error) {
	site, err := sites.FromContext(r.Context())
	if err != nil {
		return nil, err
	}
	return lang.NewManager(site, "en-US")
}

func (h *Handler) languageList(w http.ResponseWriter, r *http.Request) {
	mgr, err := h.langMgr(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	files := mgr.Files()
	data := listPage(r, 1, int64(len(files)), languagesBase,
		"Manage locale translation files and strings.",
		"", map[string]any{"ActiveNav": "languages"})
	col, dir, _ := parseSort(r, map[string]string{
		"locale": "locale", "file": "file", "scope": "scope", "keys": "keys",
	}, "locale")
	data["Sort"] = col
	data["Dir"] = dir
	sortLanguageFiles(files, col, dir)
	data["Files"] = files
	h.render(w, r, "Languages", "admin/languages.html", data)
}

func sortLanguageFiles(files []lang.LanguageFile, col, dir string) {
	sort.Slice(files, func(i, j int) bool {
		switch col {
		case "file":
			return sortLess(files[i].Filename, files[j].Filename, dir)
		case "scope":
			return sortLess(files[i].Label, files[j].Label, dir)
		case "keys":
			return sortLessInt(files[i].Count, files[j].Count, dir)
		default:
			return sortLess(files[i].Locale, files[j].Locale, dir)
		}
	})
}

func (h *Handler) languageFile(w http.ResponseWriter, r *http.Request, scope string) {
	mgr, err := h.langMgr(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if !mgr.ScopeExists(scope) {
		h.notFound(w, r)
		return
	}

	var fileInfo lang.LanguageFile
	for _, f := range mgr.Files() {
		if f.Scope == scope {
			fileInfo = f
			break
		}
	}

	if r.Method == http.MethodPost {
		if err := r.ParseForm(); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		sections, err := parseLanguageForm(r)
		if err != nil {
			h.renderLanguageFile(w, r, fileInfo, mgr.Sections(scope), err.Error())
			return
		}
		if err := mgr.ReplaceScope(scope, sections); err != nil {
			h.renderLanguageFile(w, r, fileInfo, mgr.Sections(scope), err.Error())
			return
		}
		h.invalidateLangCache(r)
		redirectList(w, r, languagesBase+"/"+scope)
		return
	}

	h.renderLanguageFile(w, r, fileInfo, mgr.Sections(scope), "")
}

func (h *Handler) renderLanguageFile(w http.ResponseWriter, r *http.Request, file lang.LanguageFile, sections []lang.LanguageSection, errMsg string) {
	title := file.Label + " Translations"
	if file.Filename != "" {
		title = file.Filename
	}
	data := formData(map[string]any{
		"ActiveNav": "languages",
		"File":      file,
		"Sections":  sections,
		"BasePath":  languagesBase,
		"Subtitle":  "Edit all sections and keys in this locale file.",
	})
	if errMsg != "" {
		data["Error"] = errMsg
	}
	h.render(w, r, title, "admin/languages_file.html", data)
}

func parseLanguageForm(r *http.Request) (map[string]map[string]string, error) {
	sections := r.Form["section"]
	keys := r.Form["key"]
	values := r.Form["value"]
	removed := map[string]bool{}
	for _, v := range r.Form["remove"] {
		removed[v] = true
	}

	out := map[string]map[string]string{}
	for i := range sections {
		if i >= len(keys) {
			break
		}
		sec := strings.TrimSpace(sections[i])
		key := strings.TrimSpace(keys[i])
		if key == "" {
			continue
		}
		removeID := sec + "|" + key
		if removed[removeID] {
			continue
		}
		val := ""
		if i < len(values) {
			val = values[i]
		}
		if out[sec] == nil {
			out[sec] = map[string]string{}
		}
		out[sec][key] = val
	}
	return out, nil
}

func (h *Handler) invalidateLangCache(r *http.Request) {
	site, err := sites.FromContext(r.Context())
	if err != nil {
		return
	}
	h.chain.InvalidateLang(site)
}
