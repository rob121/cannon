package admin

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/rob121/cannon/internal/auth"
	"github.com/rob121/cannon/internal/config"
	"github.com/rob121/cannon/internal/database"
	"github.com/rob121/cannon/internal/groups"
	"github.com/rob121/cannon/internal/lang"
	"github.com/rob121/cannon/internal/models"
	"github.com/rob121/cannon/internal/roles"
	"github.com/rob121/cannon/internal/router"
	sitectx "github.com/rob121/cannon/internal/sites"
	"gorm.io/gorm"
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
				"BasePath": sitesBase, "FormAction": siteFormAction(row, isNew),
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
		"FormAction": siteFormAction(row, isNew),
	}))
}

func siteFormAction(row config.SiteConfig, isNew bool) string {
	if isNew {
		return sitesBase + "/new"
	}
	return sitesBase + "/" + row.ID
}

func (h *Handler) saveSite(cfg *config.App, row *config.SiteConfig, isNew bool, r *http.Request) error {
	name := formString(r, "name")
	host := formString(r, "host")
	if host == "" {
		return fmt.Errorf("host URL is required")
	}
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
		if err := validateUniqueSiteHost(cfg, id, host); err != nil {
			return err
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
		if err := validateUniqueSiteHost(cfg, row.ID, host); err != nil {
			return err
		}
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
		if source := adminCopySourceSite(cfg, row, r); source != nil {
			if err := copyAdminUsersToSite(source, row); err != nil {
				return err
			}
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

func adminCopySourceSite(cfg *config.App, target *config.SiteConfig, r *http.Request) *config.SiteConfig {
	if target == nil {
		return nil
	}
	if r != nil {
		if source, err := sitectx.FromContext(r.Context()); err == nil && source != nil && source.ID != target.ID {
			return source
		}
	}
	if cfg != nil {
		for i := range cfg.Sites {
			if cfg.Sites[i].ID != target.ID {
				return &cfg.Sites[i]
			}
		}
	}
	return nil
}

func copyAdminUsersToSite(sourceSite, targetSite *config.SiteConfig) error {
	if sourceSite == nil || targetSite == nil || sourceSite.ID == targetSite.ID {
		return nil
	}
	sourceDB, err := database.Get(sourceSite.ID)
	if err != nil {
		return fmt.Errorf("source admin users: %w", err)
	}
	targetDB, err := database.Get(targetSite.ID)
	if err != nil {
		return fmt.Errorf("target admin users: %w", err)
	}

	var sourceAdminGroup models.Group
	if err := sourceDB.Where("name = ?", groups.AdministratorsGroupName).First(&sourceAdminGroup).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
		return err
	}

	var admins []models.User
	if err := sourceDB.
		Joins("JOIN user_groups ON user_groups.user_user_id = users.user_id").
		Where("user_groups.group_group_id = ? AND users.hash <> ''", sourceAdminGroup.GroupID).
		Find(&admins).Error; err != nil {
		return err
	}
	if len(admins) == 0 {
		return nil
	}

	var localAuth models.Authenticator
	if err := targetDB.Where("name = ?", "local").First(&localAuth).Error; err != nil {
		return err
	}
	var targetGroups []models.Group
	if err := targetDB.Where("name IN ?", []string{
		groups.PublicGroupName,
		groups.RegisteredGroupName,
		groups.AdministratorsGroupName,
	}).Find(&targetGroups).Error; err != nil {
		return err
	}

	for _, admin := range admins {
		copied := models.User{
			GivenName:    admin.GivenName,
			FamilyName:   admin.FamilyName,
			Email:        admin.Email,
			Username:     admin.Username,
			AvatarURL:    admin.AvatarURL,
			SSOAvatarURL: admin.SSOAvatarURL,
			Locked:       admin.Locked,
			Validated:    admin.Validated,
			Hash:         admin.Hash,
			Status:       admin.Status,
			AuthID:       &localAuth.AuthID,
		}
		var existing models.User
		err := targetDB.Where("username = ?", admin.Username).First(&existing).Error
		if err == nil {
			copied.UserID = existing.UserID
			if err := targetDB.Save(&copied).Error; err != nil {
				return err
			}
		} else if errors.Is(err, gorm.ErrRecordNotFound) {
			if err := targetDB.Create(&copied).Error; err != nil {
				return err
			}
		} else {
			return err
		}
		if len(targetGroups) > 0 {
			if err := targetDB.Model(&copied).Association("Groups").Append(targetGroups); err != nil {
				return err
			}
		}
	}
	return nil
}

func validateUniqueSiteHost(cfg *config.App, excludeID, host string) error {
	keys, err := siteHostKeys(host)
	if err != nil {
		return err
	}
	for _, site := range cfg.Sites {
		if site.ID == excludeID {
			continue
		}
		existingKeys, err := siteHostKeys(site.Host)
		if err != nil {
			return fmt.Errorf("site %s host: %w", site.ID, err)
		}
		for key := range keys {
			if _, ok := existingKeys[key]; ok {
				return fmt.Errorf("host URL conflicts with site %q", site.ID)
			}
		}
	}
	return nil
}

func siteHostKeys(raw string) (map[string]struct{}, error) {
	key, err := siteHostKey(raw)
	if err != nil {
		return nil, err
	}
	keys := map[string]struct{}{key: {}}
	if alias, ok := siteLocalHostAlias(key); ok {
		keys[alias] = struct{}{}
	}
	return keys, nil
}

func siteHostKey(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", fmt.Errorf("host URL is required")
	}
	if !strings.Contains(raw, "://") {
		raw = "https://" + raw
	}
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" {
		return "", fmt.Errorf("invalid host URL")
	}
	return siteNormalizeHost(u.Host), nil
}

func siteNormalizeHost(host string) string {
	host = strings.TrimSpace(host)
	host = strings.TrimPrefix(host, "https://")
	host = strings.TrimPrefix(host, "http://")
	if idx := strings.Index(host, "/"); idx >= 0 {
		host = host[:idx]
	}
	if idx := strings.Index(host, ":"); idx >= 0 {
		host = host[:idx]
	}
	return strings.ToLower(host)
}

func siteLocalHostAlias(host string) (string, bool) {
	switch host {
	case "localhost":
		return "127.0.0.1", true
	case "127.0.0.1":
		return "localhost", true
	default:
		return "", false
	}
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
