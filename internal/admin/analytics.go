package admin

import (
	"encoding/json"
	"html/template"
	"net/http"
	"strings"

	"github.com/rob121/cannon/internal/realtime"
	"github.com/rob121/cannon/internal/settings"
	"github.com/rob121/cannon/internal/sites"
)

func (h *Handler) analytics(w http.ResponseWriter, r *http.Request, path string) {
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) == 2 && parts[0] == "analytics" && parts[1] == "live" && r.Method == http.MethodGet {
		h.analyticsLive(w, r)
		return
	}
	h.notFound(w, r)
}

func (h *Handler) analyticsLive(w http.ResponseWriter, r *http.Request) {
	if h.realtime == nil {
		http.Error(w, "realtime unavailable", http.StatusServiceUnavailable)
		return
	}
	enabled, err := settings.AnalyticsEnabled(r.Context())
	if err != nil || !enabled {
		http.NotFound(w, r)
		return
	}
	site, err := sites.FromContext(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	stats, err := h.realtime.StatsForSite(site.ID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	_ = json.NewEncoder(w).Encode(stats)
}

func dashboardAnalyticsData(h *Handler, r *http.Request, data map[string]any) {
	enabled, err := settings.AnalyticsEnabled(r.Context())
	if err != nil || !enabled || h.realtime == nil {
		return
	}
	site, err := sites.FromContext(r.Context())
	if err != nil {
		return
	}
	data["RealtimeEnabled"] = true
	scheme := "http"
	if r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https" {
		scheme = "https"
	}
	if raw, err := realtime.ConfigJSON(site.ID, scheme, r.Host, true); err == nil {
		data["RealtimeConfigJSON"] = template.JS(raw)
	}
	if stats, err := h.realtime.StatsForSite(site.ID); err == nil {
		data["VisitorsOnline"] = stats.Online
		data["VisitorsAuthenticated"] = stats.Authenticated
		data["AnalyticsPages"] = stats.Pages
	}
}
