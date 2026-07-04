package api

import (
	"encoding/json"
	"net/http"
)

// ErrorBody is the standard API error envelope.
type ErrorBody struct {
	Error struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

// PageMeta describes list pagination.
type PageMeta struct {
	Page     int   `json:"page"`
	PageSize int   `json:"page_size"`
	Total    int64 `json:"total"`
}

// ListResponse wraps paginated data.
type ListResponse struct {
	Data any      `json:"data"`
	Meta PageMeta `json:"meta"`
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, code int, errCode, message string) {
	var body ErrorBody
	body.Error.Code = errCode
	body.Error.Message = message
	writeJSON(w, code, body)
}

func decodeJSON(r *http.Request, dst any) error {
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	return dec.Decode(dst)
}
