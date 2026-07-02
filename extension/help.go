package extension

import (
	"io/fs"
	"net/http"
	"path/filepath"
	"sort"
	"strings"
)

type helpResponse struct {
	Help []string `json:"help"`
}

type helpSource struct {
	fsys fs.FS
	root string
	base string
}

func newHelpSource(fsys fs.FS, root, base string) *helpSource {
	if base == "" {
		base = "/help"
	}
	base = strings.TrimRight(base, "/")
	return &helpSource{fsys: fsys, root: strings.Trim(root, "/"), base: base}
}

func (h *helpSource) register(mux *http.ServeMux) {
	mux.HandleFunc(h.base, h.handleIndex)
	mux.HandleFunc(h.base+"/", h.handleArticle)
}

func (h *helpSource) path() string {
	return h.base
}

func (h *helpSource) handleIndex(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != h.base {
		http.NotFound(w, r)
		return
	}
	writeJSON(w, http.StatusOK, helpResponse{Help: h.entries()})
}

func (h *helpSource) handleArticle(w http.ResponseWriter, r *http.Request) {
	name := strings.TrimPrefix(r.URL.Path, h.base+"/")
	if name == "" || strings.Contains(name, "/") {
		http.NotFound(w, r)
		return
	}
	path := filepath.Join(h.root, name+".md")
	body, err := fs.ReadFile(h.fsys, path)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "text/markdown; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(body)
}

func (h *helpSource) entries() []string {
	dir := h.root
	if dir == "" {
		dir = "."
	}
	entries, err := fs.ReadDir(h.fsys, dir)
	if err != nil {
		return nil
	}
	paths := make([]string, 0, len(entries))
	overview := h.base + "/overview"
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".md" {
			continue
		}
		slug := strings.TrimSuffix(entry.Name(), ".md")
		paths = append(paths, h.base+"/"+slug)
	}
	sort.SliceStable(paths, func(i, j int) bool {
		if paths[i] == overview {
			return true
		}
		if paths[j] == overview {
			return false
		}
		return paths[i] < paths[j]
	})
	return paths
}
