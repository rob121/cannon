package admin

import (
	"net/http"
	"strings"

	"github.com/rob121/cannon/internal/auth"
	"github.com/rob121/cannon/internal/models"
	"github.com/rob121/cannon/internal/sites"
)

const authenticatorsBase = "/admin/authenticators"

func (h *Handler) authenticators(w http.ResponseWriter, r *http.Request, path string) {
	parts := pathParts("/authenticators", path)
	switch {
	case len(parts) == 0:
		h.authenticatorList(w, r)
	case len(parts) == 2 && parts[1] == "toggle-status":
		h.authenticatorToggleStatus(w, r, parts[0])
	default:
		id, ok := parseID(parts[0])
		if !ok {
			http.NotFound(w, r)
			return
		}
		h.authenticatorForm(w, r, id)
	}
}

func (h *Handler) authenticatorList(w http.ResponseWriter, r *http.Request) {
	db, _ := sites.DB(r.Context())
	page := parsePage(r)
	var rows []models.Authenticator
	var total int64
	db.Model(&models.Authenticator{}).Count(&total)
	data := listPage(page, total, authenticatorsBase,
		"Authentication providers configured for this site.",
		"", map[string]any{"ActiveNav": "authenticators"})
	order := applyListSort(r, data, map[string]string{
		"name": "name", "status": "status",
	}, "name")
	db.Offset((page - 1) * pageSize).Limit(pageSize).Order(order).Find(&rows)
	data["Rows"] = rows
	h.render(w, r, "Authenticators", "admin/authenticators.html", data)
}

func (h *Handler) authenticatorForm(w http.ResponseWriter, r *http.Request, id uint) {
	db, _ := sites.DB(r.Context())
	var row models.Authenticator
	if err := db.First(&row, id).Error; err != nil {
		http.NotFound(w, r)
		return
	}
	if r.Method == http.MethodPost {
		if err := r.ParseForm(); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		row.Status = formStatus(r)
		row.Configuration = auth.ConfigFromForm(r, row.Name)
		if err := db.Save(&row).Error; err != nil {
			h.renderAuthenticatorForm(w, r, row, err.Error())
			return
		}
		redirectList(w, r, authenticatorsBase)
		return
	}
	normalized := auth.NormalizedConfigJSON(row.Name, row.Configuration)
	if normalized != row.Configuration {
		row.Configuration = normalized
		_ = db.Model(&row).Update("configuration", normalized).Error
	}
	h.renderAuthenticatorForm(w, r, row, "")
}

func (h *Handler) renderAuthenticatorForm(w http.ResponseWriter, r *http.Request, row models.Authenticator, errMsg string) {
	title := authenticatorTitle(row.Name)
	subtitle := "Configure Goth provider credentials and callback settings."
	data := formData(map[string]any{
		"ActiveNav":     "authenticators",
		"Row":           row,
		"BasePath":      authenticatorsBase,
		"Subtitle":      subtitle,
		"ConfigFields":  auth.ConfigFormFields(row.Name, row.Configuration),
		"ProviderLabel": displayProviderName(row.Name),
	})
	if errMsg != "" {
		data["Error"] = errMsg
	}
	h.render(w, r, title, "admin/authenticators_form.html", data)
}

func authenticatorTitle(name string) string {
	if name == "" {
		return "Edit Authenticator"
	}
	return "Edit " + displayProviderName(name)
}

func displayProviderName(name string) string {
	switch name {
	case "auth0":
		return "Auth0"
	case "azureadv2":
		return "Azure AD V2"
	case "openid":
		return "OpenID Connect"
	case "microsoftonline":
		return "Microsoft Online"
	case "wecom":
		return "WeCom"
	case "local":
		return "Local"
	}
	if len(name) == 0 {
		return name
	}
	return strings.ToUpper(name[:1]) + name[1:]
}
