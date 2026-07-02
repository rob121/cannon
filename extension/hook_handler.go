package extension

import (
	"net/http"
)

func (s *Server) registerHookRoutes(mux *http.ServeMux) {
	if len(s.hookHandlers) == 0 {
		return
	}
	base := s.hookPath
	mux.HandleFunc(base, s.handleHooks)
	mux.HandleFunc(base+"/", s.handleHooks)
}

func (s *Server) handleHooks(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.handleHooksList(w, r)
	case http.MethodPost:
		s.handleHooksDispatch(w, r)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleHooksList(w http.ResponseWriter, r *http.Request) {
	names := make([]string, 0, len(s.hookHandlers))
	for name := range s.hookHandlers {
		names = append(names, name)
	}
	writeJSON(w, http.StatusOK, HookListResponse{Hooks: names})
}

func (s *Server) handleHooksDispatch(w http.ResponseWriter, r *http.Request) {
	req, err := DecodeHookWireRequest(r, s.siteID)
	if err != nil {
		WriteWireResponse(w, Error(http.StatusBadRequest, err.Error()))
		return
	}
	fn, ok := s.hookHandlers[req.Event]
	if !ok {
		WriteWireResponse(w, Error(http.StatusNotFound, "hook not registered"))
		return
	}
	out := fn(req)
	writeHookWireResponse(w, out)
}

func writeHookWireResponse(w http.ResponseWriter, resp HookWireResponse) {
	if resp.StatusCode == 0 {
		resp.StatusCode = http.StatusOK
	}
	writeJSON(w, http.StatusOK, resp)
}
