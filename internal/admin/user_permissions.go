package admin

import (
	"context"
	"sort"

	"github.com/rob121/cannon/internal/models"
	"github.com/rob121/cannon/internal/security"
	"github.com/rob121/cannon/internal/sites"
)

// EffectivePermissionRow is one resolved permission for display.
type EffectivePermissionRow struct {
	ID          string
	DisplayName string
	Category    string
	Wildcard    bool
	Denied      bool
}

// EffectivePermissionPreview groups resolved permissions by category for the user form.
type EffectivePermissionPreview struct {
	Total       int
	AllowTotal  int
	DenyTotal   int
	Categories  []string
	ByCategory  map[string][]EffectivePermissionRow
}

func effectivePermissionPreview(ctx context.Context, userID uint) (EffectivePermissionPreview, error) {
	out := EffectivePermissionPreview{ByCategory: map[string][]EffectivePermissionRow{}}
	if userID == 0 {
		return out, nil
	}
	effective, err := security.ResolveEffective(ctx, userID)
	if err != nil {
		return out, err
	}
	labels := permissionLabelIndex(ctx)
	rows := make([]EffectivePermissionRow, 0, len(effective.Allow)+len(effective.Deny))
	for key := range effective.Deny {
		rows = append(rows, buildPermissionPreviewRow(key, labels, true))
	}
	for key := range effective.Allow {
		rows = append(rows, buildPermissionPreviewRow(key, labels, false))
	}
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].Denied != rows[j].Denied {
			return !rows[i].Denied
		}
		if rows[i].Category == rows[j].Category {
			return rows[i].ID < rows[j].ID
		}
		return rows[i].Category < rows[j].Category
	})
	for _, row := range rows {
		out.ByCategory[row.Category] = append(out.ByCategory[row.Category], row)
		if row.Denied {
			out.DenyTotal++
		} else {
			out.AllowTotal++
		}
	}
	out.Categories = make([]string, 0, len(out.ByCategory))
	for category := range out.ByCategory {
		out.Categories = append(out.Categories, category)
	}
	sort.Strings(out.Categories)
	out.Total = len(rows)
	return out, nil
}

func buildPermissionPreviewRow(key string, labels map[string]security.Permission, denied bool) EffectivePermissionRow {
	row := EffectivePermissionRow{ID: key, Category: "Other", Denied: denied}
	if key == "*" || key == security.PermWildcardAll {
		row.DisplayName = "All Permissions"
		row.Category = "Wildcard"
		row.Wildcard = true
	} else if meta, ok := labels[key]; ok {
		row.DisplayName = meta.DisplayName
		row.Category = meta.Category
		row.Wildcard = isWildcardGrant(key)
	} else {
		row.DisplayName = key
		row.Wildcard = isWildcardGrant(key)
	}
	if denied && row.DisplayName != "" && row.DisplayName != key {
		row.DisplayName += " (denied)"
	} else if denied {
		row.DisplayName = key + " (denied)"
	}
	if row.DisplayName == "" {
		row.DisplayName = key
	}
	if row.Category == "" {
		row.Category = "Other"
	}
	return row
}

func permissionLabelIndex(ctx context.Context) map[string]security.Permission {
	labels := map[string]security.Permission{}
	for _, p := range security.RegisteredPermissions() {
		labels[p.ID] = p
	}
	db, err := sites.DB(ctx)
	if err != nil {
		return labels
	}
	var rows []models.Permission
	db.Select("key", "display_name", "description", "category", "dangerous").Find(&rows)
	for _, row := range rows {
		if _, ok := labels[row.Key]; ok {
			continue
		}
		labels[row.Key] = security.Permission{
			ID:          row.Key,
			DisplayName: row.DisplayName,
			Description: row.Description,
			Category:    row.Category,
			Dangerous:   row.Dangerous,
		}
	}
	return labels
}

func isWildcardGrant(key string) bool {
	return key == "*" || key == security.PermWildcardAll || len(key) > 2 && key[len(key)-2:] == ".*"
}
