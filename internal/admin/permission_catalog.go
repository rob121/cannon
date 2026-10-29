package admin

import (
	"context"
	"sort"

	"github.com/rob121/cannon/internal/models"
	"github.com/rob121/cannon/internal/security"
	"github.com/rob121/cannon/internal/sites"
)

func permissionsForRoleAssignment(ctx context.Context, roleID uint) ([]security.Permission, error) {
	db, err := sites.DB(ctx)
	if err != nil {
		return nil, err
	}
	registered := security.RegisteredPermissions()
	filtered, err := security.FilterPermissionsByInstalledExtensions(db, registered)
	if err != nil {
		return nil, err
	}
	byID := map[string]security.Permission{}
	for _, p := range filtered {
		byID[p.ID] = p
	}
	if roleID != 0 {
		allows, denies, err := security.LoadRolePermissionAssignments(db, roleID)
		if err != nil {
			return nil, err
		}
		for _, key := range append(allows, denies...) {
			if key == "" {
				continue
			}
			if _, ok := byID[key]; ok {
				continue
			}
			if p, ok := security.PermissionByID(key); ok {
				byID[key] = p
				continue
			}
			var row models.Permission
			if err := db.Where("key = ?", key).First(&row).Error; err == nil {
				byID[key] = security.Permission{
					ID:          row.Key,
					DisplayName: row.DisplayName,
					Description: row.Description,
					Category:    row.Category,
					Dangerous:   row.Dangerous,
				}
				continue
			}
			byID[key] = security.Permission{ID: key, DisplayName: key, Category: "Other"}
		}
	}
	out := make([]security.Permission, 0, len(byID))
	for _, p := range byID {
		out = append(out, p)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Category == out[j].Category {
			return out[i].ID < out[j].ID
		}
		return out[i].Category < out[j].Category
	})
	return out, nil
}

func filterRolePermissionFormKeys(ctx context.Context, roleID uint, allowKeys, denyKeys []string) ([]string, []string) {
	allowed, err := permissionsForRoleAssignment(ctx, roleID)
	if err != nil {
		return allowKeys, denyKeys
	}
	set := map[string]struct{}{}
	for _, p := range allowed {
		set[p.ID] = struct{}{}
	}
	filter := func(keys []string) []string {
		out := make([]string, 0, len(keys))
		for _, key := range keys {
			if key == "" {
				continue
			}
			if _, ok := set[key]; ok {
				out = append(out, key)
			}
		}
		return out
	}
	return filter(allowKeys), filter(denyKeys)
}

func permissionsForCatalog(ctx context.Context) ([]security.Permission, error) {
	db, err := sites.DB(ctx)
	if err != nil {
		return nil, err
	}
	registered := security.RegisteredPermissions()
	filtered, err := security.FilterPermissionsByInstalledExtensions(db, registered)
	if err != nil {
		return nil, err
	}
	byKey := map[string]security.Permission{}
	for _, p := range filtered {
		byKey[p.ID] = p
	}
	var dbPerms []models.Permission
	if err := db.Where("deprecated = ?", false).Order("category asc, key asc").Find(&dbPerms).Error; err != nil {
		return nil, err
	}
	for _, row := range dbPerms {
		if _, ok := byKey[row.Key]; ok {
			continue
		}
		ext, isExt := security.PermissionExtensionOwner(row.Key)
		if isExt {
			installed, err := security.InstalledExtensionNames(db)
			if err != nil {
				return nil, err
			}
			if _, ok := installed[ext]; !ok {
				continue
			}
		}
		byKey[row.Key] = security.Permission{
			ID:          row.Key,
			DisplayName: row.DisplayName,
			Description: row.Description,
			Category:    row.Category,
			Dangerous:   row.Dangerous,
		}
	}
	out := make([]security.Permission, 0, len(byKey))
	for _, p := range byKey {
		out = append(out, p)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Category == out[j].Category {
			return out[i].ID < out[j].ID
		}
		return out[i].Category < out[j].Category
	})
	return out, nil
}
