package httpx

import "net/http"

// Redirect sends a temporary redirect that browsers must not cache.
// Use this instead of http.Redirect for install and auth flows.
func Redirect(w http.ResponseWriter, r *http.Request, url string) {
	w.Header().Set("Cache-Control", "no-store, no-cache, must-revalidate")
	w.Header().Set("Pragma", "no-cache")
	http.Redirect(w, r, url, http.StatusFound)
}

// RedirectSeeOther sends a 303 redirect (POST → GET) that browsers must not cache.
func RedirectSeeOther(w http.ResponseWriter, r *http.Request, url string) {
	w.Header().Set("Cache-Control", "no-store, no-cache, must-revalidate")
	w.Header().Set("Pragma", "no-cache")
	http.Redirect(w, r, url, http.StatusSeeOther)
}
