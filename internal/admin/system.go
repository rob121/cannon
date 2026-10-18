package admin

import (
	"net/http"
	"strconv"

	"github.com/rob121/cannon/internal/accesslog"
	"github.com/rob121/cannon/internal/blocks"
	"github.com/rob121/cannon/internal/sites"
)

const systemAccessLogBase = "/admin/system/access-log"

func (h *Handler) system(w http.ResponseWriter, r *http.Request, path string) {
	parts := pathParts("/system", path)
	switch {
	case len(parts) == 1 && parts[0] == "reload":
		h.systemReload(w, r)
	case len(parts) == 1 && parts[0] == "update":
		h.systemUpdate(w, r)
	case len(parts) == 2 && parts[0] == "update" && parts[1] == "status":
		h.systemUpdateStatus(w, r)
	case len(parts) >= 1 && parts[0] == "access-log":
		if len(parts) == 2 && parts[1] == "tail" {
			h.systemAccessLogTail(w, r)
			return
		}
		h.systemAccessLog(w, r)
	default:
		h.notFound(w, r)
	}
}

func (h *Handler) systemAccessLog(w http.ResponseWriter, r *http.Request) {
	site, err := sites.FromContext(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	files, err := accesslog.Files(site)
	if err != nil {
		h.render(w, r, "Access Log", "admin/system_access_log.html", formData(map[string]any{
			"ActiveNav": "access_log",
			"Subtitle":  "HTTP request log for this site.",
			"Error":     err.Error(),
		}))
		return
	}
	selected := r.URL.Query().Get("file")
	if selected == "" && len(files) > 0 {
		selected = files[0].Name
	}
	var active accesslog.File
	for _, file := range files {
		if file.Name == selected {
			active = file
			break
		}
	}
	content := ""
	if active.Path != "" {
		content, err = accesslog.Tail(active.Path, 128*1024)
		if err != nil {
			h.render(w, r, "Access Log", "admin/system_access_log.html", formData(map[string]any{
				"ActiveNav": "access_log",
				"Subtitle":  "HTTP request log for this site.",
				"Error":     err.Error(),
			}))
			return
		}
	}
	h.render(w, r, "Access Log", "admin/system_access_log.html", formData(map[string]any{
		"ActiveNav":    "access_log",
		"Subtitle":     "HTTP request log for this site.",
		"LogPath":      accesslog.Path(site),
		"LogHostKey":   accesslog.HostKey(site.Host),
		"Files":        files,
		"SelectedFile": selected,
		"LogContent":   content,
		"TailURL":      systemAccessLogBase + "/tail",
	}))
}

func (h *Handler) systemAccessLogTail(w http.ResponseWriter, r *http.Request) {
	site, err := sites.FromContext(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	file, err := accesslog.ResolveFile(site, r.URL.Query().Get("file"))
	if err != nil {
		http.Error(w, "log file not found", http.StatusNotFound)
		return
	}
	maxBytes := int64(128 * 1024)
	if raw := r.URL.Query().Get("bytes"); raw != "" {
		if n, err := strconv.ParseInt(raw, 10, 64); err == nil && n > 0 {
			maxBytes = n
		}
	}
	content, err := accesslog.Tail(file.Path, maxBytes)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	_, _ = w.Write([]byte(content))
}

func (h *Handler) systemReload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if h.reload == nil {
		http.Error(w, "reload unavailable", http.StatusInternalServerError)
		return
	}
	data := map[string]any{
		"ActiveNav": "dashboard",
		"Subtitle":  dashboardWelcomeSubtitle(r),
	}
	if err := h.reload(); err != nil {
		data["Error"] = "Reload failed: " + err.Error()
	} else {
		data["Success"] = "Application reloaded. Routes, configuration, and extensions were refreshed without restarting the process."
	}
	h.renderDashboard(w, r, data)
}

func (h *Handler) renderDashboard(w http.ResponseWriter, r *http.Request, data map[string]any) {
	_ = blocks.EnsureAdminDashboardBlocks(r.Context())
	if _, ok := data["Subtitle"]; !ok {
		data["Subtitle"] = dashboardWelcomeSubtitle(r)
	}
	h.render(w, r, "Dashboard", "admin/dashboard.html", data)
}
