package extension

import (
	"net/http"
	"sort"
	"strings"
)

type endpointEntry struct {
	def EndpointDefinition
	fn  EndpointHandler
}

func (s *Server) registerEndpointRoutes(mux *http.ServeMux) {
	if len(s.endpoints) == 0 {
		return
	}
	base := s.endpointBase()
	mux.HandleFunc(base, s.handleEndpointIndex)
	mux.HandleFunc(base+"/", s.handleEndpointItem)
}

func (s *Server) handleEndpointIndex(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != s.endpointBase() {
		http.NotFound(w, r)
		return
	}
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	writeJSON(w, http.StatusOK, s.endpointList())
}

func (s *Server) handleEndpointItem(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	item := strings.TrimPrefix(r.URL.Path, s.endpointBase()+"/")
	item = strings.Trim(item, "/")
	if item == "" || strings.Contains(item, "/") {
		http.NotFound(w, r)
		return
	}
	entry, ok := s.endpoints[item]
	if !ok {
		http.NotFound(w, r)
		return
	}
	req, err := DecodeWireRequest(r, s.siteID)
	if err != nil {
		WriteWireResponse(w, Error(http.StatusBadRequest, err.Error()))
		return
	}
	if req.EndpointItem == "" {
		req.EndpointItem = item
	}
	WriteWireResponse(w, entry.fn(item, req))
}

func (s *Server) endpointList() EndpointListResponse {
	if s.endpointListProvider != nil {
		items, err := s.endpointListProvider()
		if err == nil {
			sort.Slice(items, func(i, j int) bool { return items[i].ID < items[j].ID })
			return EndpointListResponse{Endpoints: items}
		}
	}
	items := make([]EndpointDefinition, 0, len(s.endpoints))
	for _, entry := range s.endpoints {
		items = append(items, entry.def)
	}
	sort.Slice(items, func(i, j int) bool { return items[i].ID < items[j].ID })
	return EndpointListResponse{Endpoints: items}
}

// EndpointItem returns the endpoint definition id being handled.
func EndpointItem(req WireRequest) string {
	return strings.TrimSpace(req.EndpointItem)
}
