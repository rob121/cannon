package admin

import (
	"net/http"
	"net/url"
	"strings"
)

// SafeExtensionAdminRedirectLocation validates an extension admin redirect target.
// Only same-origin URLs under /admin/extension-apps/{name} are allowed.
func SafeExtensionAdminRedirectLocation(extensionName, location string, r *http.Request) (string, bool) {
	location = strings.TrimSpace(location)
	if location == "" || r == nil || r.URL == nil {
		return "", false
	}
	name := strings.TrimSpace(extensionName)
	if name == "" {
		return "", false
	}
	basePath := extensionAppsBase + "/" + url.PathEscape(name)

	if strings.HasPrefix(location, "/") && !strings.HasPrefix(location, "//") {
		if location == basePath || strings.HasPrefix(location, basePath+"/") || strings.HasPrefix(location, basePath+"?") {
			return location, true
		}
		return "", false
	}

	u, err := url.Parse(location)
	if err != nil {
		return "", false
	}
	if u.Scheme == "" {
		ref, err := r.URL.Parse(location)
		if err != nil {
			return "", false
		}
		return SafeExtensionAdminRedirectLocation(name, ref.String(), r)
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return "", false
	}
	if !strings.EqualFold(u.Host, r.URL.Host) {
		return "", false
	}
	path := u.EscapedPath()
	if path == "" {
		path = "/"
	}
	if path == basePath || strings.HasPrefix(path, basePath+"/") {
		return u.String(), true
	}
	return "", false
}
