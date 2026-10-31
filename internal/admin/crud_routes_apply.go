package admin

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/rob121/cannon/extension"
	"github.com/rob121/cannon/internal/extensions"
	"github.com/rob121/cannon/internal/groups"
	"github.com/rob121/cannon/internal/httpx"
	"github.com/rob121/cannon/internal/models"
	"github.com/rob121/cannon/internal/routemeta"
	"github.com/rob121/cannon/internal/router"
	"github.com/rob121/cannon/internal/sites"
	"gorm.io/gorm"
)

func extensionRouteApplyOptions(extMgr *extensions.Manager, db *gorm.DB) []extensionRouteApplyOption {
	menuNames := extensionMenuNames(db)
	out := make([]extensionRouteApplyOption, 0)
	seen := map[string]struct{}{}
	for _, rt := range extMgr.PageRuntimes() {
		if len(rt.PageMounts) == 0 && len(rt.PageRoutes) == 0 {
			continue
		}
		if _, ok := seen[rt.Model.Name]; ok {
			continue
		}
		if !extensionRouteBundleNeedsApply(db, rt.Model.Name, rt.PageMounts, rt.PageRoutes) {
			continue
		}
		seen[rt.Model.Name] = struct{}{}
		out = append(out, extensionRouteApplyOption{
			Name:        rt.Model.Name,
			Label:       extensionMenuLabel(rt.Model.Name, menuNames),
			Mounts:      append([]extension.MountSpec(nil), rt.PageMounts...),
			FixedRoutes: append([]extension.PageRouteSpec(nil), rt.PageRoutes...),
		})
	}
	return out
}

func extensionRouteBundleNeedsApply(db *gorm.DB, extName string, mounts []extension.MountSpec, fixed []extension.PageRouteSpec) bool {
	for _, mount := range mounts {
		if !mount.Resolver {
			continue
		}
		path := normalizeExtensionMountPath(normalizeRouteSpecPath(mount.Path))
		if path == "" {
			continue
		}
		if !extensionRouteApplied(db, extName, path, "") {
			return true
		}
	}
	for _, spec := range fixed {
		path := normalizeRouteSpecPath(spec.Path)
		pageID := strings.TrimSpace(spec.PageID)
		if path == "" || pageID == "" {
			continue
		}
		if !extensionRouteApplied(db, extName, path, pageID) {
			return true
		}
	}
	return false
}

func extensionRouteApplied(db *gorm.DB, extName, path, pageID string) bool {
	q := db.Model(&models.Route{}).Where("path = ? AND extension_name = ?", path, extName)
	if pageID == "" {
		q = q.Where("extension_page_id = ''")
	} else {
		q = q.Where("extension_page_id = ?", pageID)
	}
	var count int64
	q.Count(&count)
	return count > 0
}

func (h *Handler) routeApplyExtensionRoutes(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	extName := strings.TrimSpace(formString(r, "extension_name"))
	if extName == "" {
		http.Error(w, "extension_name is required", http.StatusBadRequest)
		return
	}
	skipExisting := formBool(r, "skip_existing")

	db, err := sites.DB(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	site, err := sites.FromContext(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	extMgr := h.chain.Extensions(site)
	_ = extMgr.Bootstrap(r.Context())
	extMgr.EnsurePageDefinitions(r.Context())

	rt, ok := extMgr.Runtime(extName)
	if !ok {
		http.Error(w, "extension not found or not running", http.StatusBadRequest)
		return
	}
	mounts := rt.PageMounts
	fixed := rt.PageRoutes
	if len(mounts) == 0 && len(fixed) == 0 {
		http.Error(w, "extension has no declared mounts or routes", http.StatusBadRequest)
		return
	}

	created := 0
	skipped := 0
	var applyErr error
	applyErr = db.Transaction(func(tx *gorm.DB) error {
		var maxSort int
		_ = tx.Model(&models.Route{}).Select("COALESCE(MAX(sort), -1)").Scan(&maxSort)

		for _, mount := range mounts {
			path := normalizeExtensionMountPath(normalizeRouteSpecPath(mount.Path))
			if path == "" {
				continue
			}
			if skipExisting && routePathTaken(tx, path) {
				if err := ensureExistingRoutePublicGroup(tx, path); err != nil {
					return err
				}
				skipped++
				continue
			}
			if router.ConflictsWithReservedPath(path) {
				return fmt.Errorf("path %s conflicts with a reserved route", path)
			}
			meta := ""
			for k, v := range mount.DefaultPageData {
				var err error
				meta, err = routemeta.SetMetadataString(meta, k, v)
				if err != nil {
					return err
				}
			}
			if mount.ID != "" {
				var err error
				meta, err = routemeta.SetMetadataString(meta, "extension_mount_id", mount.ID)
				if err != nil {
					return err
				}
			}
			name := strings.TrimSpace(mount.Title)
			if name == "" {
				name = mount.ID
			}
			if name == "" {
				name = extName
			}
			row := models.Route{
				Name:            name,
				Path:            path,
				Type:            models.RouteTypeExtension,
				Status:          models.StatusActive,
				ExtensionName:   extName,
				ExtensionPageID: "",
				Metadata:        meta,
				ShowTitle:       false,
				Sort:            maxSort + 1,
			}
			if !mount.Resolver {
				skipped++
				continue
			}
			if err := tx.Create(&row).Error; err != nil {
				if skipExisting && strings.Contains(strings.ToLower(err.Error()), "unique") {
					skipped++
					continue
				}
				return err
			}
			if err := assignRoutePublicGroup(tx, &row); err != nil {
				return err
			}
			maxSort++
			created++
		}

		for _, spec := range fixed {
			path := normalizeRouteSpecPath(spec.Path)
			pageID := strings.TrimSpace(spec.PageID)
			if path == "" || pageID == "" {
				continue
			}
			if skipExisting && routePathTaken(tx, path) {
				if err := ensureExistingRoutePublicGroup(tx, path); err != nil {
					return err
				}
				skipped++
				continue
			}
			if router.ConflictsWithReservedPath(path) {
				return fmt.Errorf("path %s conflicts with a reserved route", path)
			}
			name := strings.TrimSpace(spec.Name)
			if name == "" {
				name = pageID
			}
			row := models.Route{
				Name:            name,
				Path:            path,
				Type:            models.RouteTypeExtension,
				Status:          models.StatusActive,
				ExtensionName:   extName,
				ExtensionPageID: pageID,
				ShowTitle:       true,
				Sort:            maxSort + 1,
			}
			if err := tx.Create(&row).Error; err != nil {
				if skipExisting && strings.Contains(strings.ToLower(err.Error()), "unique") {
					skipped++
					continue
				}
				return err
			}
			if err := assignRoutePublicGroup(tx, &row); err != nil {
				return err
			}
			maxSort++
			created++
		}
		return nil
	})
	if applyErr != nil {
		http.Error(w, applyErr.Error(), http.StatusBadRequest)
		return
	}
	invalidateRoutesDataCache(r.Context())
	dest := routesBase + "?applied=1&created=" + fmt.Sprintf("%d", created) + "&skipped=" + fmt.Sprintf("%d", skipped)
	httpx.Redirect(w, r, dest)
}

func normalizeRouteSpecPath(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return ""
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	return path
}

func routePathTaken(db *gorm.DB, path string) bool {
	var count int64
	db.Model(&models.Route{}).Where("path = ?", path).Count(&count)
	return count > 0
}

func ensureExistingRoutePublicGroup(db *gorm.DB, path string) error {
	var row models.Route
	if err := db.Preload("Groups").Where("path = ?", path).First(&row).Error; err != nil {
		return err
	}
	if len(row.Groups) > 0 {
		return nil
	}
	return assignRoutePublicGroup(db, &row)
}

func assignRoutePublicGroup(db *gorm.DB, route *models.Route) error {
	if route == nil || route.RouteID == 0 {
		return fmt.Errorf("route must be saved before assigning groups")
	}
	publicID, err := groups.PublicGroupID(db)
	if err != nil {
		return err
	}
	var publicGroup models.Group
	if err := db.First(&publicGroup, publicID).Error; err != nil {
		return err
	}
	return db.Model(route).Association("Groups").Replace([]models.Group{publicGroup})
}
