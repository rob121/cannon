package extension

import (
	"net/http"
	"sort"
	"strings"
)

type pageEntry struct {
	def PageDefinition
	fn  PageHandler
}

func (s *Server) registerPageRoutes(mux *http.ServeMux) {
	if len(s.pages) == 0 && s.pageResolveHandler == nil {
		return
	}
	base := s.pageBase()
	mux.HandleFunc(base, s.handlePageIndex)
	if s.pageResolveHandler != nil {
		mux.HandleFunc(base+"/resolve", s.handlePageResolve)
	}
	if len(s.pages) > 0 {
		mux.HandleFunc(base+"/", s.handlePageItem)
	}
}

func (s *Server) handlePageResolve(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if s.pageResolveHandler == nil {
		http.NotFound(w, r)
		return
	}
	req, err := DecodeWireRequest(r, s.siteID)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	out := s.pageResolveHandler(req)
	if out.StatusCode == 0 {
		out.StatusCode = http.StatusOK
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *Server) handlePageIndex(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != s.pageBase() {
		http.NotFound(w, r)
		return
	}
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	writeJSON(w, http.StatusOK, s.pageList())
}

func (s *Server) handlePageItem(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	item := strings.TrimPrefix(r.URL.Path, s.pageBase()+"/")
	item = strings.Trim(item, "/")
	if item == "" || strings.Contains(item, "/") {
		http.NotFound(w, r)
		return
	}
	entry, ok := s.pages[item]
	if !ok {
		http.NotFound(w, r)
		return
	}
	req, err := DecodeWireRequest(r, s.siteID)
	if err != nil {
		WriteWireResponse(w, Error(http.StatusBadRequest, err.Error()))
		return
	}
	if req.PageItem == "" {
		req.PageItem = item
	}
	WriteWireResponse(w, entry.fn(item, req))
}

func (s *Server) pageList() PageListResponse {
	if s.pageCatalogProvider != nil {
		catalog, err := s.pageCatalogProvider()
		if err == nil {
			sort.Slice(catalog.Pages, func(i, j int) bool { return catalog.Pages[i].ID < catalog.Pages[j].ID })
			return catalog
		}
	}
	if s.pageListProvider != nil {
		items, err := s.pageListProvider()
		if err == nil {
			sort.Slice(items, func(i, j int) bool { return items[i].ID < items[j].ID })
			return PageListResponse{Pages: items}
		}
	}
	items := make([]PageDefinition, 0, len(s.pages))
	for _, entry := range s.pages {
		items = append(items, entry.def)
	}
	sort.Slice(items, func(i, j int) bool { return items[i].ID < items[j].ID })
	return PageListResponse{Pages: items}
}

// PageItem returns the page definition id being rendered.
func PageItem(req WireRequest) string {
	return strings.TrimSpace(req.PageItem)
}
