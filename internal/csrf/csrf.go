package csrf

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"html"
	"html/template"
	"net/http"
	"strings"
)

const (
	// SessionKey stores the token in session data.
	SessionKey = "csrf_token"
	// FieldName is the form field extensions and templates should use.
	FieldName = "_csrf"
	// HeaderName is the request header for non-form submissions.
	HeaderName = "X-CSRF-Token"
)

// ErrInvalid is returned when a CSRF token is missing or does not match.
var ErrInvalid = errors.New("invalid csrf token")

// GenerateToken returns a new random CSRF token.
func GenerateToken() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}

// IsMutating reports whether an HTTP method changes server state.
func IsMutating(method string) bool {
	switch strings.ToUpper(strings.TrimSpace(method)) {
	case http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete:
		return true
	default:
		return false
	}
}

// SubmittedToken reads a token from the header or form body.
func SubmittedToken(r *http.Request) string {
	if v := strings.TrimSpace(r.Header.Get(HeaderName)); v != "" {
		return v
	}
	if r.Form == nil {
		_ = r.ParseForm()
	}
	return strings.TrimSpace(r.FormValue(FieldName))
}

// Valid compares expected and submitted tokens in constant time.
func Valid(expected, submitted string) bool {
	if expected == "" || submitted == "" {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(expected), []byte(submitted)) == 1
}

// HiddenField returns HTML for a hidden CSRF input.
func HiddenField(token string) template.HTML {
	if token == "" {
		return template.HTML("")
	}
	return template.HTML(`<input type="hidden" name="` + FieldName + `" value="` + html.EscapeString(token) + `">`)
}
