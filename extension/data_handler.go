package extension

import (
	"net/http"
	"strings"
)

func (s *Server) registerDataRoutes(mux *http.ServeMux) {
	if len(s.dataHandlers) == 0 && s.dataFallback == nil {
		return
	}
	base := s.dataBase()
	mux.HandleFunc(base, s.handleData)
	mux.HandleFunc(base+"/", s.handleData)
}

func (s *Server) handleData(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	req, err := DecodeWireRequest(r, s.siteID)
	if err != nil {
		WriteWireResponse(w, Error(http.StatusBadRequest, err.Error()))
		return
	}

	rel := strings.TrimPrefix(r.URL.Path, s.dataBase())
	rel = strings.Trim(rel, "/")
	if rel == "" {
		rel = DataPath(req)
	}
	if rel != "" {
		req.DataPath = rel
	}

	fn, ok := s.dataHandlers[rel]
	if !ok && s.dataFallback != nil {
		fn = s.dataFallback
		ok = true
	}
	if !ok {
		WriteWireResponse(w, Error(http.StatusNotFound, "data route not found"))
		return
	}
	WriteWireResponse(w, fn(req))
}
