package templateengine

import (
	"fmt"
	"io"
	"net/http"
	"strings"
)

const (
	errorPagePartial    = "default/partials/error-page.html"
	errorFallbackPage   = "default/error.html"
	adminErrorPartial   = "admin/error/_page.html"
	adminErrorFallback  = "admin/error.html"
)

// ErrorTitle returns a human-friendly title for an HTTP status code.
func ErrorTitle(code int) string {
	switch code {
	case http.StatusNotFound:
		return "Not Found"
	case http.StatusForbidden:
		return "Access Denied"
	case http.StatusBadRequest:
		return "Bad Request"
	case http.StatusUnauthorized:
		return "Unauthorized"
	case http.StatusInternalServerError:
		return "Server Error"
	case http.StatusServiceUnavailable:
		return "Service Unavailable"
	default:
		return "Error"
	}
}

// DefaultErrorMessage returns a generic message when none is supplied.
func DefaultErrorMessage(code int) string {
	switch code {
	case http.StatusNotFound:
		return "The page you requested could not be found."
	case http.StatusForbidden:
		return "You do not have permission to view this page."
	case http.StatusBadRequest:
		return "The request could not be processed."
	case http.StatusUnauthorized:
		return "Sign in is required to view this page."
	case http.StatusInternalServerError:
		return "Something went wrong while processing your request."
	case http.StatusServiceUnavailable:
		return "This site is temporarily unavailable."
	default:
		return "An error occurred."
	}
}

// ErrorTemplatePath returns the conventional frontend error template path for a status code.
func ErrorTemplatePath(code int) string {
	return fmt.Sprintf("default/error/%d.html", code)
}

// AdminErrorTemplatePath returns the conventional admin error template path for a status code.
func AdminErrorTemplatePath(code int) string {
	return fmt.Sprintf("admin/error/%d.html", code)
}

// ResolveErrorTemplate picks a frontend error page, falling back to default/error.html.
func (e *Engine) ResolveErrorTemplate(code int) string {
	if e.templateExists(ErrorTemplatePath(code)) {
		return ErrorTemplatePath(code)
	}
	if e.templateExists(errorFallbackPage) {
		return errorFallbackPage
	}
	return ErrorTemplatePath(code)
}

// ResolveAdminErrorTemplate picks an admin error page, falling back to admin/error.html.
func (e *Engine) ResolveAdminErrorTemplate(code int) string {
	if e.templateExists(AdminErrorTemplatePath(code)) {
		return AdminErrorTemplatePath(code)
	}
	if e.templateExists(adminErrorFallback) {
		return adminErrorFallback
	}
	return AdminErrorTemplatePath(code)
}

// RenderError renders a frontend error page inside the site layout.
func (e *Engine) RenderError(w io.Writer, code int, data map[string]any) error {
	if data == nil {
		data = map[string]any{}
	}
	if _, ok := data["Title"]; !ok {
		data["Title"] = ErrorTitle(code)
	}
	if _, ok := data["ErrorCode"]; !ok {
		data["ErrorCode"] = code
	}
	if msg, ok := data["ErrorMessage"].(string); !ok || strings.TrimSpace(msg) == "" {
		data["ErrorMessage"] = DefaultErrorMessage(code)
	}
	page := e.ResolveErrorTemplate(code)
	return e.render(w, "default/layout.html", page, data, code)
}

// RenderAdminError renders an admin error page inside the admin layout.
func (e *Engine) RenderAdminError(w io.Writer, code int, data map[string]any) error {
	if data == nil {
		data = map[string]any{}
	}
	if _, ok := data["Title"]; !ok {
		data["Title"] = ErrorTitle(code)
	}
	page := e.ResolveAdminErrorTemplate(code)
	return e.render(w, "admin/layout.html", page, data, code)
}

func (e *Engine) templateExists(name string) bool {
	_, _, err := e.readTemplate(name)
	return err == nil
}
