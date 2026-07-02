package extensions

import (
	"net/http"
	"strings"
)

// WriteHTTPResponse writes a wire response to the client with status, headers, and body.
func WriteHTTPResponse(w http.ResponseWriter, wire WireResponse) {
	status := wire.StatusCode
	if status == 0 {
		status = http.StatusOK
	}
	for key, vals := range wire.Header {
		for _, val := range vals {
			w.Header().Add(key, val)
		}
	}
	if wire.Body != "" && len(wire.Header["Content-Type"]) == 0 && len(wire.Header["content-type"]) == 0 {
		if strings.HasPrefix(strings.TrimSpace(wire.Body), "{") || strings.HasPrefix(strings.TrimSpace(wire.Body), "[") {
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
		} else if strings.Contains(wire.Body, "<") {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
		} else {
			w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		}
	}
	w.WriteHeader(status)
	if wire.Body != "" {
		_, _ = w.Write([]byte(wire.Body))
	}
}
