package templateengine

import (
	"html/template"
	"net/http"

	"github.com/rob121/cannon/internal/csrf"
	"github.com/rob121/cannon/internal/user"
)

// CSRFFuncs returns template helpers for the current request session.
func CSRFFuncs(r *http.Request) template.FuncMap {
	return template.FuncMap{
		"csrfToken": func() string {
			return csrfTokenFromRequest(r)
		},
		"csrfField": func() template.HTML {
			return csrf.HiddenField(csrfTokenFromRequest(r))
		},
	}
}

func csrfTokenFromRequest(r *http.Request) string {
	svc, err := user.FromContext(r.Context())
	if err != nil {
		return ""
	}
	token, err := svc.EnsureCSRFToken()
	if err != nil {
		return ""
	}
	return token
}

// MergeFuncMaps combines multiple FuncMaps; later maps override earlier keys.
func MergeFuncMaps(maps ...template.FuncMap) template.FuncMap {
	out := template.FuncMap{}
	for _, m := range maps {
		for k, v := range m {
			out[k] = v
		}
	}
	return out
}
