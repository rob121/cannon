package extension

import "html"

// CSRFToken returns the Cannon session CSRF token attached to the wire request.
func CSRFToken(req WireRequest) string {
	return req.CSRF
}

// CSRFHiddenField returns HTML for a hidden CSRF input suitable for extension-rendered forms.
func CSRFHiddenField(req WireRequest) string {
	if req.CSRF == "" {
		return ""
	}
	return `<input type="hidden" name="_csrf" value="` + html.EscapeString(req.CSRF) + `">`
}
