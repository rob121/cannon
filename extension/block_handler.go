package extension

import (
	"net/http"
	"sort"
	"strings"
)

type blockEntry struct {
	def BlockDefinition
	fn  BlockHandler
}

func (s *Server) registerBlockRoutes(mux *http.ServeMux) {
	if len(s.blocks) == 0 {
		return
	}
	base := s.blockBase()
	mux.HandleFunc(base, s.handleBlockIndex)
	mux.HandleFunc(base+"/", s.handleBlockItem)
}

func (s *Server) handleBlockIndex(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != s.blockBase() {
		http.NotFound(w, r)
		return
	}
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	writeJSON(w, http.StatusOK, s.blockList())
}

func (s *Server) handleBlockItem(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	item := strings.TrimPrefix(r.URL.Path, s.blockBase()+"/")
	item = strings.Trim(item, "/")
	if item == "" || strings.Contains(item, "/") {
		http.NotFound(w, r)
		return
	}
	entry, ok := s.blocks[item]
	if !ok {
		http.NotFound(w, r)
		return
	}
	req, err := DecodeWireRequest(r, s.siteID)
	if err != nil {
		WriteWireResponse(w, Error(http.StatusBadRequest, err.Error()))
		return
	}
	if req.BlockItem == "" {
		req.BlockItem = item
	}
	WriteWireResponse(w, entry.fn(item, req))
}

func (s *Server) blockList() BlockListResponse {
	if s.blockListProvider != nil {
		items, err := s.blockListProvider()
		if err == nil {
			sort.Slice(items, func(i, j int) bool { return items[i].ID < items[j].ID })
			return BlockListResponse{Blocks: items}
		}
	}
	items := make([]BlockDefinition, 0, len(s.blocks))
	for _, entry := range s.blocks {
		items = append(items, entry.def)
	}
	sort.Slice(items, func(i, j int) bool { return items[i].ID < items[j].ID })
	return BlockListResponse{Blocks: items}
}

// BlockSpace returns the template space Cannon is rendering.
func BlockSpace(req WireRequest) string {
	return strings.TrimSpace(req.BlockSpace)
}

// BlockItem returns the block definition id being rendered.
func BlockItem(req WireRequest) string {
	return strings.TrimSpace(req.BlockItem)
}
