package groups

import (
	"context"

	"github.com/rob121/cannon/internal/models"
	"github.com/rob121/cannon/internal/sites"
	"github.com/rob121/cannon/internal/user"
	"gorm.io/gorm"
)

const PublicGroupName = "public"
const RegisteredGroupName = "registered"

const (
	AdministratorsGroupName = "Administrators"
	ManagerGroupName        = "Manager"
	EditorGroupName         = "Editor"
	WriterGroupName         = "Writer"
)

// ViewerGroupIDs returns the active group ids that apply to the current request viewer.
// Anonymous visitors receive only the public group. Signed-in users receive their
// assigned groups plus public.
func ViewerGroupIDs(ctx context.Context) ([]uint, error) {
	db, err := sites.DB(ctx)
	if err != nil {
		return nil, err
	}
	publicID, err := PublicGroupID(db)
	if err != nil {
		return nil, err
	}

	svc, err := user.FromContext(ctx)
	if err != nil {
		return []uint{publicID}, nil
	}
	userID, ok := svc.CurrentID()
	if !ok {
		return []uint{publicID}, nil
	}

	var ids []uint
	err = db.Model(&models.User{}).
		Joins("JOIN user_groups ON user_groups.user_user_id = users.user_id").
		Joins("JOIN groups ON groups.group_id = user_groups.group_group_id").
		Where("users.user_id = ? AND users.status = ? AND groups.status = ?",
			userID, models.StatusActive, models.StatusActive).
		Pluck("groups.group_id", &ids).Error
	if err != nil {
		return nil, err
	}
	return ensureContains(ids, publicID), nil
}

// CanView reports whether content assigned to contentGroups is visible to viewerGroupIDs.
func CanView(viewerGroupIDs []uint, contentGroups []models.Group) bool {
	active := activeGroups(contentGroups)
	if len(active) == 0 {
		return false
	}
	contentIDs := groupIDs(active)
	for _, viewerID := range viewerGroupIDs {
		for _, contentID := range contentIDs {
			if viewerID == contentID {
				return true
			}
		}
	}
	return false
}

// PublicGroupID returns the id of the hardcoded public group.
func PublicGroupID(db *gorm.DB) (uint, error) {
	var group models.Group
	if err := db.Where("name = ?", PublicGroupName).First(&group).Error; err != nil {
		return 0, err
	}
	return group.GroupID, nil
}

// EnsurePublicGroup seeds the hardcoded public group used for content visibility.
func EnsurePublicGroup(db *gorm.DB) error {
	var group models.Group
	if err := db.Where("name = ?", PublicGroupName).First(&group).Error; err == gorm.ErrRecordNotFound {
		group = models.Group{Name: PublicGroupName, Kind: models.GroupKindFrontend, Status: models.StatusActive}
		return db.Create(&group).Error
	} else if err != nil {
		return err
	}
	return db.Model(&group).Update("kind", models.GroupKindFrontend).Error
}

// EnsureDefaults seeds public/registered groups and backfills content visibility defaults.
func EnsureDefaults(db *gorm.DB) error {
	if err := EnsurePublicGroup(db); err != nil {
		return err
	}
	public, err := publicGroup(db)
	if err != nil {
		return err
	}
	if err := ensureRegisteredGroup(db, public.GroupID); err != nil {
		return err
	}
	if err := ensureUserPublicMembership(db, public); err != nil {
		return err
	}
	return ensureContentPublicDefaults(db, public)
}

func ensureRegisteredGroup(db *gorm.DB, publicID uint) error {
	var group models.Group
	err := db.Where("name = ?", RegisteredGroupName).First(&group).Error
	if err == gorm.ErrRecordNotFound {
		group = models.Group{
			Name:     RegisteredGroupName,
			Kind:     models.GroupKindFrontend,
			ParentID: &publicID,
			Status:   models.StatusActive,
		}
		return db.Create(&group).Error
	}
	if err != nil {
		return err
	}
	return db.Model(&group).Updates(map[string]any{
		"kind":      models.GroupKindFrontend,
		"parent_id": publicID,
	}).Error
}

func publicGroup(db *gorm.DB) (models.Group, error) {
	id, err := PublicGroupID(db)
	if err != nil {
		return models.Group{}, err
	}
	var group models.Group
	if err := db.First(&group, id).Error; err != nil {
		return models.Group{}, err
	}
	return group, nil
}

func ensureUserPublicMembership(db *gorm.DB, public models.Group) error {
	var users []models.User
	if err := db.Find(&users).Error; err != nil {
		return err
	}
	for _, u := range users {
		var count int64
		if err := db.Table("user_groups").
			Where("user_user_id = ? AND group_group_id = ?", u.UserID, public.GroupID).
			Count(&count).Error; err != nil {
			return err
		}
		if count == 0 {
			if err := db.Model(&u).Association("Groups").Append(&public); err != nil {
				return err
			}
		}
	}
	return nil
}

func ensureContentPublicDefaults(db *gorm.DB, public models.Group) error {
	var blocks []models.Block
	if err := db.Find(&blocks).Error; err != nil {
		return err
	}
	for _, row := range blocks {
		if err := ensureRowPublicGroup(db, "block_groups", "block_block_id", row.BlockID, public, &row); err != nil {
			return err
		}
	}

	var routes []models.Route
	if err := db.Find(&routes).Error; err != nil {
		return err
	}
	for _, row := range routes {
		if err := ensureRowPublicGroup(db, "route_groups", "route_route_id", row.RouteID, public, &row); err != nil {
			return err
		}
	}

	var items []models.MenuItem
	if err := db.Find(&items).Error; err != nil {
		return err
	}
	for _, row := range items {
		if err := ensureRowPublicGroup(db, "menu_item_groups", "menu_item_menu_item_id", row.MenuItemID, public, &row); err != nil {
			return err
		}
	}
	return nil
}

func ensureRowPublicGroup(db *gorm.DB, table, idColumn string, id uint, public models.Group, model any) error {
	var count int64
	if err := db.Table(table).Where(idColumn+" = ?", id).Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return nil
	}
	return db.Model(model).Association("Groups").Replace([]models.Group{public})
}

// AssignRegistered adds a user to the registered frontend group.
func AssignRegistered(ctx context.Context, userID uint) error {
	db, err := sites.DB(ctx)
	if err != nil {
		return err
	}
	var group models.Group
	if err := db.Where("name = ?", RegisteredGroupName).First(&group).Error; err != nil {
		return err
	}
	var u models.User
	if err := db.First(&u, userID).Error; err != nil {
		return err
	}
	return db.Model(&u).Association("Groups").Append(&group)
}

// AssignPublic adds a user to the public group.
func AssignPublic(ctx context.Context, userID uint) error {
	db, err := sites.DB(ctx)
	if err != nil {
		return err
	}
	var group models.Group
	if err := db.Where("name = ?", PublicGroupName).First(&group).Error; err != nil {
		return err
	}
	var u models.User
	if err := db.First(&u, userID).Error; err != nil {
		return err
	}
	return db.Model(&u).Association("Groups").Append(&group)
}

func activeGroups(groups []models.Group) []models.Group {
	out := make([]models.Group, 0, len(groups))
	for _, group := range groups {
		if group.Status == models.StatusActive {
			out = append(out, group)
		}
	}
	return out
}

func groupIDs(groups []models.Group) []uint {
	out := make([]uint, 0, len(groups))
	for _, group := range groups {
		out = append(out, group.GroupID)
	}
	return out
}

func ensureContains(ids []uint, id uint) []uint {
	for _, existing := range ids {
		if existing == id {
			return ids
		}
	}
	return append(ids, id)
}
