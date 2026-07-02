package extension

import (
	"encoding/json"
	"net/http"
)

// WireRequest is the JSON payload Cannon POSTs to capability handlers.
type WireRequest struct {
	Method string              `json:"method"`
	URL    string              `json:"url"`
	Header map[string][]string `json:"header"`
	Body   string              `json:"body"`
	User   map[string]any      `json:"user,omitempty"` // signed-in user scope from Cannon session
	CSRF   string              `json:"csrf,omitempty"` // session CSRF token for forms and mutating requests
	SiteID string              `json:"site_id"`
	BlockSpace string          `json:"block_space,omitempty"`
	BlockItem  string          `json:"block_item,omitempty"`
	BlockData  map[string]any  `json:"block_data,omitempty"`
	PageItem   string          `json:"page_item,omitempty"`
	PageData   map[string]any  `json:"page_data,omitempty"`
	EndpointItem string        `json:"endpoint_item,omitempty"`
	EndpointData map[string]any `json:"endpoint_data,omitempty"`
	DataPath     string        `json:"data_path,omitempty"`
}

// WireResponse is the JSON payload capability handlers return to Cannon.
type WireResponse struct {
	StatusCode int                 `json:"status_code"`
	Header     map[string][]string `json:"header"`
	Body       string              `json:"body"`
	Updated    *WireRequest        `json:"updated_request,omitempty"`
	Stop       bool                `json:"stop"`
}

// Handler handles a Cannon wire request.
type Handler func(WireRequest) WireResponse

// InstallFunc runs one-time extension setup during POST /install.
type InstallFunc func(req WireRequest) error

// HTML returns a wire response with an HTML body.
func HTML(status int, body string) WireResponse {
	if status == 0 {
		status = http.StatusOK
	}
	return WireResponse{
		StatusCode: status,
		Header: map[string][]string{
			"Content-Type": {"text/html; charset=utf-8"},
		},
		Body: body,
	}
}

// OK returns a successful empty wire response.
func OK() WireResponse {
	return WireResponse{StatusCode: http.StatusOK}
}

// Error returns a wire response describing an error.
func Error(status int, message string) WireResponse {
	if status == 0 {
		status = http.StatusInternalServerError
	}
	return WireResponse{
		StatusCode: status,
		Body:       message,
		Stop:       true,
	}
}

// Redirect returns a wire response that redirects the browser.
func Redirect(status int, location string) WireResponse {
	if status == 0 {
		status = http.StatusFound
	}
	return WireResponse{
		StatusCode: status,
		Header: map[string][]string{
			"Location": {location},
		},
	}
}

// DecodeWireRequest reads a wire request from an HTTP request body.
func DecodeWireRequest(r *http.Request, fallbackSiteID string) (WireRequest, error) {
	var req WireRequest
	if r.Body != nil && r.ContentLength != 0 {
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			return WireRequest{}, err
		}
	}
	if req.SiteID == "" {
		req.SiteID = fallbackSiteID
	}
	return req, nil
}

// WriteWireResponse writes a wire response as JSON.
func WriteWireResponse(w http.ResponseWriter, resp WireResponse) {
	if resp.StatusCode == 0 {
		resp.StatusCode = http.StatusOK
	}
	writeJSON(w, http.StatusOK, resp)
}
