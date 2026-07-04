package extension

import (
	"fmt"
	"net/http"
	"strings"
)

func (s *Server) registerCaptchaRoutes(mux *http.ServeMux) {
	if s.captcha == nil {
		return
	}
	base := s.captchaPath
	mux.HandleFunc(base, s.handleCaptcha)
	mux.HandleFunc(base+"/", s.handleCaptchaSubpath)
}

func (s *Server) handleCaptcha(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != s.captchaPath && r.URL.Path != s.captchaPath+"/" {
		http.NotFound(w, r)
		return
	}
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	req, err := DecodeWireRequest(r, s.siteID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	info, err := s.captchaProviderInfo(req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, info)
}

func (s *Server) handleCaptchaSubpath(w http.ResponseWriter, r *http.Request) {
	action := strings.TrimPrefix(r.URL.Path, strings.TrimRight(s.captchaPath, "/")+"/")
	action = strings.Trim(action, "/")
	switch action {
	case "render":
		s.handleCaptchaRender(w, r)
	case "verify":
		s.handleCaptchaVerify(w, r)
	default:
		http.NotFound(w, r)
	}
}

func (s *Server) handleCaptchaRender(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if s.captcha.Render == nil {
		http.Error(w, "captcha render not configured", http.StatusNotImplemented)
		return
	}
	req, err := DecodeWireRequest(r, s.siteID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if CaptchaContext(req) == "" {
		http.Error(w, "captcha_context is required", http.StatusBadRequest)
		return
	}
	out, err := s.captcha.Render(req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if strings.TrimSpace(out.FieldName) == "" {
		http.Error(w, "captcha render must set field_name", http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *Server) handleCaptchaVerify(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if s.captcha.Verify == nil {
		http.Error(w, "captcha verify not configured", http.StatusNotImplemented)
		return
	}
	req, err := DecodeWireRequest(r, s.siteID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if CaptchaContext(req) == "" {
		http.Error(w, "captcha_context is required", http.StatusBadRequest)
		return
	}
	if CaptchaToken(req) == "" {
		http.Error(w, "captcha_token is required", http.StatusBadRequest)
		return
	}
	out, err := s.captcha.Verify(req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	status := http.StatusOK
	if !out.Valid {
		status = http.StatusForbidden
	}
	writeJSON(w, status, out)
}

func (s *Server) captchaProviderInfo(req WireRequest) (CaptchaProviderInfo, error) {
	if s.captcha.ProviderInfo != nil {
		return s.captcha.ProviderInfo(req)
	}
	id := strings.TrimSpace(s.info.Name)
	if id == "" {
		return CaptchaProviderInfo{}, fmt.Errorf("captcha provider id is required")
	}
	title := strings.TrimSpace(s.info.Title)
	if title == "" {
		title = id
	}
	return CaptchaProviderInfo{
		ID:    id,
		Title: title,
		Contexts: []string{
			CaptchaContextLogin,
			CaptchaContextRegister,
			CaptchaContextComment,
			CaptchaContextForm,
		},
	}, nil
}
