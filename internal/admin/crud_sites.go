package admin

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sort"

	"github.com/rob121/cannon/internal/auth"
	"github.com/rob121/cannon/internal/config"
	"github.com/rob121/cannon/internal/database"
	"github.com/rob121/cannon/internal/lang"
	"github.com/rob121/cannon/internal/roles"
	"github.com/rob121/cannon/internal/router"
)

const sitesBase = "/admin/sites"

func (h *Handler) sites(w http.ResponseWriter, r *http.Request, path string) {
	parts := pathParts("/sites", path)
	switch {
	case len(parts) == 0:
		h.siteList(w, r)
	case parts[0] == "new":
		h.siteForm(w, r, "")
	case len(parts) == 2 && parts[1] == "delete":
		h.siteDelete(w, r, parts[0])
	default:
		h.siteForm(w, r, parts[0])
	}
}

func (h *Handler) siteList(w http.ResponseWriter, r *http.Request) {
	cfg := h.chain.Sites.Config()
	sites := append([]config.SiteConfig(nil), cfg.Sites...)
	data := listPage(r, 1, int64(len(sites)), sitesBase,
		"Multi-site configuration and host mappings.",
		"Add Site", map[string]any{"ActiveNav": "sites"})
	col, dir, _ := parseSort(r, map[string]string{
		"id": "id", "name": "name", "host": "host",
	}, "name")
	data["Sort"] = col
	data["Dir"] = dir
	sortSiteConfigs(sites, col, dir)
	data["Sites"] = sites
	h.render(w, r, "Sites", "admin/sites.html", data)
}

func sortSiteConfigs(sites []config.SiteConfig, col, dir string) {
	sort.Slice(sites, func(i, j int) bool {
		switch col {
		case "id":
			return sortLess(sites[i].ID, sites[j].ID, dir)
		case "host":
			return sortLess(sites[i].Host, sites[j].Host, dir)
		default:
			return sortLess(sites[i].Name, sites[j].Name, dir)
		}
	})
}

func (h *Handler) siteForm(w http.ResponseWriter, r *http.Request, siteID string) {
	cfg := h.chain.Sites.Config()
	isNew := siteID == ""
	var row config.SiteConfig
	if !isNew {
		found := false
		for _, s := range cfg.Sites {
			if s.ID == siteID {
				row = s
				found = true
				break
			}
		}
		if !found {
			h.notFound(w, r)
			return
		}
	}
	if r.Method == http.MethodPost {
		if err := r.ParseForm(); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if err := h.saveSite(cfg, &row, isNew, r); err != nil {
			h.render(w, r, "Site", "admin/sites_form.html", formData(map[string]any{
				"ActiveNav": "sites", "Error": err.Error(), "Row": row, "IsNew": isNew,
			}))
			return
		}
		redirectList(w, r, sitesBase)
		return
	}
	title := "Add Site"
	if !isNew {
		title = "Edit Site"
	}
	h.render(w, r, title, "admin/sites_form.html", formData(map[string]any{
		"ActiveNav": "sites", "Row": row, "IsNew": isNew, "BasePath": sitesBase,
	}))
}

func (h *Handler) saveSite(cfg *config.App, row *config.SiteConfig, isNew bool, r *http.Request) error {
	name := formString(r, "name")
	host := formString(r, "host")
	dbType := formString(r, "database_type")
	if dbType == "" {
		dbType = "sqlite"
	}
	dataRoot := cfg.DataRoot
	if dataRoot == "" {
		dataRoot = "./data"
	}

	id := row.ID
	if isNew {
		id = slugify(name)
		if id == "" {
			return fmt.Errorf("site name is required")
		}
		for _, s := range cfg.Sites {
			if s.ID == id {
				return fmt.Errorf("site id %q already exists", id)
			}
		}
		siteDir := filepath.Join(dataRoot, id)
		row.ID = id
		row.Name = name
		row.Host = host
		row.TemplateDir = filepath.Join(siteDir, "templates")
		row.AssetsDir = filepath.Join(siteDir, "assets")
		row.TmpDir = filepath.Join(siteDir, "tmp")
		row.LanguageDir = filepath.Join(siteDir, "language")
		row.Database.Type = dbType
		if dbType == "sqlite" {
			row.Database.DSN = filepath.Join(siteDir, id+".sqlite")
		} else {
			row.Database.DSN = formString(r, "database_dsn")
		}
		for _, dir := range []string{row.TemplateDir, row.AssetsDir, row.TmpDir, row.LanguageDir, siteDir} {
			if err := os.MkdirAll(dir, 0755); err != nil {
				return err
			}
		}
		if err := lang.EnsureDefaults(row.LanguageDir); err != nil {
			return err
		}
		cfg.Sites = append(cfg.Sites, *row)
	} else {
		row.Name = name
		row.Host = host
		row.Database.Type = dbType
		if dbType != "sqlite" {
			if dsn := formString(r, "database_dsn"); dsn != "" {
				row.Database.DSN = dsn
			}
		}
		for i := range cfg.Sites {
			if cfg.Sites[i].ID == row.ID {
				cfg.Sites[i] = *row
				break
			}
		}
	}

	if host == "" {
		row.Host = "http://localhost:8001"
	}

	if err := config.Save(cfg); err != nil {
		return err
	}
	if isNew {
		if _, err := database.Connect(row); err != nil {
			return err
		}
		if err := database.Migrate(row); err != nil {
			return err
		}
		db, _ := database.Get(row.ID)
		if err := auth.SeedAuthenticators(db); err != nil {
			return err
		}
		if err := roles.EnsureDefaults(db); err != nil {
			return err
		}
		if err := router.EnsureDefaultRoute(db); err != nil {
			return err
		}
	}
	if err := h.chain.Sites.Reload(cfg); err != nil {
		return err
	}
	if h.activate != nil {
		return h.activate(cfg)
	}
	return nil
}

func (h *Handler) siteDelete(w http.ResponseWriter, r *http.Request, siteID string) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	cfg := h.chain.Sites.Config()
	if len(cfg.Sites) <= 1 {
		http.Error(w, "cannot delete the last site", http.StatusBadRequest)
		return
	}
	next := make([]config.SiteConfig, 0, len(cfg.Sites)-1)
	found := false
	for _, s := range cfg.Sites {
		if s.ID == siteID {
			found = true
			continue
		}
		next = append(next, s)
	}
	if !found {
		h.notFound(w, r)
		return
	}
	cfg.Sites = next
	if err := config.Save(cfg); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if err := h.chain.Sites.Reload(cfg); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if h.activate != nil {
		_ = h.activate(cfg)
	}
	redirectList(w, r, sitesBase)
}
