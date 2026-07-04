package admin

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/rob121/cannon/internal/api"
	"github.com/rob121/cannon/internal/models"
	"github.com/rob121/cannon/internal/security"
	"github.com/rob121/cannon/internal/sites"
	"github.com/rob121/cannon/internal/user"
)

const apiCredentialsBase = "/admin/api/credentials"

func (h *Handler) apiSection(w http.ResponseWriter, r *http.Request, path string) {
	parts := pathParts("/api", path)
	if len(parts) > 0 && parts[0] == "docs" {
		http.Redirect(w, r, "/api/v1/docs", http.StatusSeeOther)
		return
	}
	if len(parts) > 0 && parts[0] == "settings" {
		h.apiSettings(w, r)
		return
	}
	if len(parts) == 0 || parts[0] == "credentials" {
		sub := []string{}
		if len(parts) > 1 {
			sub = parts[1:]
		}
		h.apiCredentials(w, r, sub)
		return
	}
	h.notFound(w, r)
}

func (h *Handler) apiCredentials(w http.ResponseWriter, r *http.Request, parts []string) {
	switch {
	case len(parts) == 0:
		h.apiCredentialList(w, r)
	case parts[0] == "new":
		h.apiCredentialForm(w, r, 0)
	case len(parts) == 2 && parts[1] == "rotate":
		h.apiCredentialRotate(w, r, parts[0])
	case len(parts) == 2 && parts[1] == "toggle-status":
		h.apiCredentialToggle(w, r, parts[0])
	default:
		id, ok := parseID(parts[0])
		if !ok {
			h.notFound(w, r)
			return
		}
		h.apiCredentialForm(w, r, id)
	}
}

func (h *Handler) apiCredentialList(w http.ResponseWriter, r *http.Request) {
	db, _ := sites.DB(r.Context())
	page := parsePage(r)
	var rows []models.APICredential
	var total int64
	db.Model(&models.APICredential{}).Count(&total)
	db.Order("credential_id desc").Offset((page - 1) * pageSizeFor(r)).Limit(pageSizeFor(r)).Find(&rows)
	data := listPage(r, page, total, apiCredentialsBase,
		"Manage API keys for headless frontends. Tokens are shown once on create or rotate.",
		"New Credential", map[string]any{"ActiveNav": "api_credentials"})
	data["Rows"] = rows
	h.render(w, r, "API Credentials", "admin/api_credentials.html", data)
}

func (h *Handler) apiCredentialForm(w http.ResponseWriter, r *http.Request, id uint) {
	db, _ := sites.DB(r.Context())
	isNew := id == 0
	var row models.APICredential
	if !isNew {
		if err := db.First(&row, id).Error; err != nil {
			h.notFound(w, r)
			return
		}
	}
	if r.Method == http.MethodPost {
		if err := r.ParseForm(); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if isNew {
			var expires *time.Time
			if raw := strings.TrimSpace(r.FormValue("expires_at")); raw != "" {
				if t, err := time.Parse("2006-01-02", raw); err == nil {
					end := t.Add(24*time.Hour - time.Second)
					expires = &end
				}
			}
			var createdBy *uint
			if svc, err := user.FromContext(r.Context()); err == nil {
				if uid, ok := svc.CurrentID(); ok {
					createdBy = &uid
				}
			}
			row, token, err := api.IssueCredential(r.Context(), formString(r, "name"), expires, createdBy)
			if err != nil {
				h.renderAPICredentialForm(w, r, row, isNew, err.Error(), "")
				return
			}
			h.renderAPICredentialForm(w, r, row, false, "", token)
			return
		}
		row.Name = formString(r, "name")
		row.Status = formStatus(r)
		if raw := strings.TrimSpace(r.FormValue("expires_at")); raw != "" {
			if t, err := time.Parse("2006-01-02", raw); err == nil {
				end := t.Add(24*time.Hour - time.Second)
				row.ExpiresAt = &end
			}
		} else {
			row.ExpiresAt = nil
		}
		if err := db.Save(&row).Error; err != nil {
			h.renderAPICredentialForm(w, r, row, isNew, err.Error(), "")
			return
		}
		redirectList(w, r, apiCredentialsBase)
		return
	}
	h.renderAPICredentialForm(w, r, row, isNew, "", "")
}

func (h *Handler) apiCredentialRotate(w http.ResponseWriter, r *http.Request, idRaw string) {
	if r.Method != http.MethodPost {
		h.notFound(w, r)
		return
	}
	id, ok := parseID(idRaw)
	if !ok {
		h.notFound(w, r)
		return
	}
	token, err := api.RotateCredential(r.Context(), id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	db, _ := sites.DB(r.Context())
	var row models.APICredential
	_ = db.First(&row, id)
	h.renderAPICredentialForm(w, r, row, false, "", token)
}

func (h *Handler) apiCredentialToggle(w http.ResponseWriter, r *http.Request, idRaw string) {
	if r.Method != http.MethodPost && r.Method != http.MethodGet {
		h.notFound(w, r)
		return
	}
	id, ok := parseID(idRaw)
	if !ok {
		h.notFound(w, r)
		return
	}
	db, _ := sites.DB(r.Context())
	var row models.APICredential
	if err := db.First(&row, id).Error; err != nil {
		h.notFound(w, r)
		return
	}
	if row.Status == models.StatusActive {
		row.Status = models.StatusInactive
	} else {
		row.Status = models.StatusActive
	}
	_ = db.Save(&row).Error
	redirectList(w, r, apiCredentialsBase)
}

func (h *Handler) renderAPICredentialForm(w http.ResponseWriter, r *http.Request, row models.APICredential, isNew bool, errMsg, newToken string) {
	data := formData(map[string]any{
		"ActiveNav": "api_credentials",
		"Row":       row,
		"IsNew":     isNew,
		"BasePath":  apiCredentialsBase,
		"Error":     errMsg,
		"NewToken":  newToken,
	})
	h.render(w, r, "API Credential", "admin/api_credentials_form.html", data)
}

func (h *Handler) apiSettings(w http.ResponseWriter, r *http.Request) {
	if !h.requireAPIWrite(r) {
		h.forbidden(w, r)
		return
	}
	ctx := r.Context()
	if r.Method == http.MethodPost {
		if err := r.ParseForm(); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		data, _ := api.SettingsData(ctx)
		if data == nil {
			data = map[string]any{}
		}
		if v := strings.TrimSpace(r.FormValue("jwt_ttl_seconds")); v != "" {
			if n, err := strconv.Atoi(v); err == nil {
				data["jwt_ttl_seconds"] = n
			}
		}
		if v := strings.TrimSpace(r.FormValue("cors_origins")); v != "" {
			data["cors_origins"] = v
		} else {
			data["cors_origins"] = ""
		}
		if v := strings.TrimSpace(r.FormValue("rate_limit_per_minute")); v != "" {
			if n, err := strconv.Atoi(v); err == nil {
				data["rate_limit_per_minute"] = n
			}
		}
		if v := strings.TrimSpace(r.FormValue("login_rate_limit_per_minute")); v != "" {
			if n, err := strconv.Atoi(v); err == nil {
				data["login_rate_limit_per_minute"] = n
			}
		}
		if r.FormValue("rotate_jwt_secret") == "1" {
			_ = api.RotateJWTSecret(ctx)
		}
		_ = api.SaveSettingsData(ctx, data)
		http.Redirect(w, r, "/admin/api/settings?saved=1", http.StatusSeeOther)
		return
	}
	data, _ := api.SettingsData(ctx)
	h.render(w, r, "API Settings", "admin/api_settings.html", map[string]any{
		"ActiveNav": "api_settings",
		"Settings":  data,
		"Saved":     r.URL.Query().Get("saved") == "1",
		"DocsURL":   "/api/v1/docs",
	})
}

func (h *Handler) requireAPIWrite(r *http.Request) bool {
	svc, err := user.FromContext(r.Context())
	if err != nil {
		return false
	}
	userID, ok := svc.CurrentID()
	if !ok {
		return false
	}
	allowed, err := security.Can(r.Context(), userID, "core.admin.api.write")
	return err == nil && allowed
}
