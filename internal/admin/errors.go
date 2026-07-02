package admin

import (
	"fmt"
	"net/http"
)

func errorIconForStatus(status int) string {
	switch status {
	case http.StatusForbidden:
		return "bi-shield-lock"
	case http.StatusNotFound:
		return "bi-search"
	case http.StatusBadRequest:
		return "bi-exclamation-triangle"
	default:
		return "bi-exclamation-circle"
	}
}

func (h *Handler) renderError(w http.ResponseWriter, r *http.Request, status int, title, message, hint, backURL string) {
	if backURL == "" {
		backURL = "/admin"
	}
	data := map[string]any{
		"Title":        title,
		"ActiveNav":    "_error",
		"Subtitle":     message,
		"ErrorStatus":  status,
		"ErrorMessage": message,
		"ErrorHint":    hint,
		"ErrorIcon":    errorIconForStatus(status),
		"BackURL":      backURL,
	}
	eng, err := h.engine(r, nil)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	eng.SetHookContext(r.Context())
	defer eng.SetHookContext(nil)
	if err := eng.RenderAdminError(w, status, data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (h *Handler) notFound(w http.ResponseWriter, r *http.Request, messages ...string) {
	message := "The page you requested could not be found."
	if len(messages) > 0 && messages[0] != "" {
		message = messages[0]
	}
	h.renderError(w, r, http.StatusNotFound, "Not Found", message,
		"Check the URL or return to the dashboard.", "/admin")
}

func (h *Handler) forbidden(w http.ResponseWriter, r *http.Request) {
	h.renderError(w, r, http.StatusForbidden, "Access Denied",
		"You do not have permission to access this admin section.",
		"Contact an administrator if you need access, or return to the dashboard.", "/admin")
}

func writeStandaloneAdminError(w http.ResponseWriter, status int, title, message, hint string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	icon := errorIconForStatus(status)
	fmt.Fprint(w, standaloneAdminErrorHTML(status, title, message, hint, icon))
}

func standaloneAdminErrorHTML(status int, title, message, hint, icon string) string {
	return fmt.Sprintf(`<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>%s - Cannon Admin</title>
  <link rel="preconnect" href="https://fonts.googleapis.com">
  <link href="https://fonts.googleapis.com/css2?family=Inter:wght@400;500;600;700&display=swap" rel="stylesheet">
  <link href="https://cdn.jsdelivr.net/npm/bootstrap@5.3.3/dist/css/bootstrap.min.css" rel="stylesheet">
  <link href="https://cdn.jsdelivr.net/npm/bootstrap-icons@1.11.3/font/bootstrap-icons.min.css" rel="stylesheet">
  <link href="/admin/assets/admin.css?v=32" rel="stylesheet">
</head>
<body class="admin-login-page">
  <div class="admin-install-wrap admin-error-standalone">
    <div class="admin-install-brand">
      <div class="admin-brand-icon"><img class="admin-cannon-icon" src="/admin/assets/cannon-icon.svg?v=4" alt="" width="32" height="32"></div>
      <div class="admin-error-icon admin-error-icon-lg" aria-hidden="true"><i class="bi %s"></i></div>
      <p class="admin-error-code">%d</p>
      <h1>%s</h1>
      <p>%s</p>
      %s
    </div>
    <div class="admin-form-card admin-install-card">
      <div class="admin-form-card-body admin-error-actions">
        <a class="btn-admin-primary w-100" href="/admin/login"><i class="bi bi-box-arrow-in-right"></i> Back to Sign In</a>
      </div>
    </div>
  </div>
</body>
</html>`, title, icon, status, title, message, hintBlock(hint))
}

func hintBlock(hint string) string {
	if hint == "" {
		return ""
	}
	return fmt.Sprintf(`<p class="admin-error-hint">%s</p>`, hint)
}
