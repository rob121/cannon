package csrf

import (
	"bytes"
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"html"
	"html/template"
	"io"
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
	if token := parsedFormFieldToken(r); token != "" {
		return token
	}
	if err := parseFormPreserveBody(r); err != nil {
		return ""
	}
	return parsedFormFieldToken(r)
}

func parsedFormFieldToken(r *http.Request) string {
	if r.MultipartForm != nil {
		if vals := r.MultipartForm.Value[FieldName]; len(vals) > 0 {
			return strings.TrimSpace(vals[0])
		}
	}
	if r.Form != nil {
		return strings.TrimSpace(r.Form.Get(FieldName))
	}
	return ""
}

func parseFormPreserveBody(r *http.Request) error {
	if r == nil {
		return ErrInvalid
	}
	if r.Body == nil {
		return r.ParseForm()
	}
	raw, err := io.ReadAll(r.Body)
	if err != nil {
		return err
	}
	r.Body = io.NopCloser(bytes.NewReader(raw))
	ct := r.Header.Get("Content-Type")
	if strings.HasPrefix(ct, "multipart/form-data") {
		err = r.ParseMultipartForm(32 << 20)
	} else {
		err = r.ParseForm()
	}
	r.Body = io.NopCloser(bytes.NewReader(raw))
	return err
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
