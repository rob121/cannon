package content

import (
	"context"
	"errors"

	"github.com/rob121/cannon/internal/models"
	"github.com/rob121/cannon/internal/settings"
	"github.com/rob121/cannon/internal/sites"
	"gorm.io/gorm"
)

const (
	settingCreateGroupIDs  = "create_group_ids"
	settingEditGroupIDs    = "edit_group_ids"
	settingPublishGroupIDs = "publish_group_ids"
)

// PermissionGroups holds resolved frontend content permission groups.
type PermissionGroups struct {
	Create  []models.Group
	Edit    []models.Group
	Publish []models.Group
}

// LoadPermissionGroupIDs reads global content permission group ids from settings.
func LoadPermissionGroupIDs(ctx context.Context) (create, edit, publish []uint, err error) {
	data, err := settings.NewStore().Load(ctx, settings.ScopeGlobal, SettingsSection)
	if err != nil {
		return nil, nil, nil, err
	}
	return uintSliceSetting(data, settingCreateGroupIDs),
		uintSliceSetting(data, settingEditGroupIDs),
		uintSliceSetting(data, settingPublishGroupIDs), nil
}

// LoadPermissionGroups loads global content permission groups from settings.
func LoadPermissionGroups(ctx context.Context) (PermissionGroups, error) {
	createIDs, editIDs, publishIDs, err := LoadPermissionGroupIDs(ctx)
	if err != nil {
		return PermissionGroups{}, err
	}
	db, err := sites.DB(ctx)
	if err != nil {
		return PermissionGroups{}, err
	}
	create, err := groupsByIDs(db, createIDs)
	if err != nil {
		return PermissionGroups{}, err
	}
	edit, err := groupsByIDs(db, editIDs)
	if err != nil {
		return PermissionGroups{}, err
	}
	publish, err := groupsByIDs(db, publishIDs)
	if err != nil {
		return PermissionGroups{}, err
	}
	return PermissionGroups{Create: create, Edit: edit, Publish: publish}, nil
}

// ResolvePermissionCategory returns the category whose permission groups apply,
// walking parents while InheritPermissions is enabled.
func ResolvePermissionCategory(ctx context.Context, cat *models.Category) (*models.Category, error) {
	if cat == nil {
		return nil, nil
	}
	db, err := sites.DB(ctx)
	if err != nil {
		return nil, err
	}
	current := cat
	for current.InheritPermissions && current.ParentID != nil && *current.ParentID > 0 {
		var parent models.Category
		if err := db.Preload("CreateGroups").Preload("EditGroups").Preload("PublishGroups").
			First(&parent, *current.ParentID).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				break
			}
			return nil, err
		}
		current = &parent
	}
	return current, nil
}

// EffectiveCreateGroups returns create groups for a category, falling back to global settings.
func EffectiveCreateGroups(ctx context.Context, categoryID *uint) ([]models.Group, error) {
	global, err := LoadPermissionGroups(ctx)
	if err != nil {
		return nil, err
	}
	if categoryID == nil || *categoryID == 0 {
		return global.Create, nil
	}
	cat, err := categoryWithPermissionGroups(ctx, *categoryID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return global.Create, nil
		}
		return nil, err
	}
	resolved, err := ResolvePermissionCategory(ctx, cat)
	if err != nil {
		return nil, err
	}
	if resolved != nil && len(resolved.CreateGroups) > 0 {
		return activePermissionGroups(resolved.CreateGroups), nil
	}
	return global.Create, nil
}

// EffectiveEditGroups returns edit groups for a category, falling back to global settings.
func EffectiveEditGroups(ctx context.Context, categoryID *uint) ([]models.Group, error) {
	global, err := LoadPermissionGroups(ctx)
	if err != nil {
		return nil, err
	}
	if categoryID == nil || *categoryID == 0 {
		return global.Edit, nil
	}
	cat, err := categoryWithPermissionGroups(ctx, *categoryID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return global.Edit, nil
		}
		return nil, err
	}
	resolved, err := ResolvePermissionCategory(ctx, cat)
	if err != nil {
		return nil, err
	}
	if resolved != nil && len(resolved.EditGroups) > 0 {
		return activePermissionGroups(resolved.EditGroups), nil
	}
	return global.Edit, nil
}

// EffectivePublishGroups returns publish groups for a category, falling back to global settings.
func EffectivePublishGroups(ctx context.Context, categoryID *uint) ([]models.Group, error) {
	global, err := LoadPermissionGroups(ctx)
	if err != nil {
		return nil, err
	}
	if categoryID == nil || *categoryID == 0 {
		return global.Publish, nil
	}
	cat, err := categoryWithPermissionGroups(ctx, *categoryID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return global.Publish, nil
		}
		return nil, err
	}
	resolved, err := ResolvePermissionCategory(ctx, cat)
	if err != nil {
		return nil, err
	}
	if resolved != nil && len(resolved.PublishGroups) > 0 {
		return activePermissionGroups(resolved.PublishGroups), nil
	}
	return global.Publish, nil
}

func categoryWithPermissionGroups(ctx context.Context, id uint) (*models.Category, error) {
	db, err := sites.DB(ctx)
	if err != nil {
		return nil, err
	}
	var cat models.Category
	if err := db.Preload("CreateGroups").Preload("EditGroups").Preload("PublishGroups").
		First(&cat, id).Error; err != nil {
		return nil, err
	}
	return &cat, nil
}

func groupsByIDs(db *gorm.DB, ids []uint) ([]models.Group, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	var groups []models.Group
	if err := db.Where("group_id IN ? AND status = ?", ids, models.StatusActive).
		Order("name asc").Find(&groups).Error; err != nil {
		return nil, err
	}
	return groups, nil
}

func activePermissionGroups(groups []models.Group) []models.Group {
	out := make([]models.Group, 0, len(groups))
	for _, group := range groups {
		if group.Status == models.StatusActive {
			out = append(out, group)
		}
	}
	return out
}

func uintSliceSetting(data map[string]any, key string) []uint {
	v, ok := data[key]
	if !ok || v == nil {
		return nil
	}
	switch arr := v.(type) {
	case []uint:
		return arr
	case []any:
		out := make([]uint, 0, len(arr))
		for _, item := range arr {
			switch n := item.(type) {
			case float64:
				if n > 0 {
					out = append(out, uint(n))
				}
			case int:
				if n > 0 {
					out = append(out, uint(n))
				}
			case uint:
				if n > 0 {
					out = append(out, n)
				}
			}
		}
		return out
	default:
		return nil
	}
}
