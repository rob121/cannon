package admin

import (
	"net/http"
	"sort"
	"strings"

	"github.com/rob121/cannon/internal/config"
	"github.com/rob121/cannon/internal/sites"
	"github.com/rob121/cannon/internal/user"
)

func (h *Handler) siteSwitcherOptions(r *http.Request) []map[string]any {
	if h == nil || h.chain == nil {
		return nil
	}
	svc, err := user.FromContext(r.Context())
	if err != nil {
		return nil
	}
	u, err := svc.Current(r.Context())
	if err != nil || u.UserID == 0 {
		return nil
	}
	ok, err := IsSuperAdmin(r.Context(), u.UserID)
	if err != nil || !ok {
		return nil
	}
	cfg := h.chain.Sites.Config()
	if cfg == nil || len(cfg.Sites) <= 1 {
		return nil
	}
	current, _ := sites.FromContext(r.Context())
	var currentID string
	if current != nil {
		currentID = current.ID
	}
	return buildSiteSwitcherOptions(cfg.Sites, currentID)
}

func buildSiteSwitcherOptions(sites []config.SiteConfig, currentID string) []map[string]any {
	opts := make([]map[string]any, 0, len(sites))
	for _, s := range sites {
		label := strings.TrimSpace(s.Name)
		if label == "" {
			label = siteHostLabel(s.Host)
		}
		opts = append(opts, map[string]any{
			"ID":        s.ID,
			"Label":     label,
			"HostLabel": siteHostLabel(s.Host),
			"AdminURL":  siteAdminURL(s.Host),
			"Current":   s.ID == currentID,
		})
	}
	sort.Slice(opts, func(i, j int) bool {
		li, _ := opts[i]["Label"].(string)
		lj, _ := opts[j]["Label"].(string)
		return strings.ToLower(li) < strings.ToLower(lj)
	})
	return opts
}
