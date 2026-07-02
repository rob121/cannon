package content

import (
	"context"
	"errors"

	"github.com/rob121/cannon/internal/groups"
	"github.com/rob121/cannon/internal/models"
	"github.com/rob121/cannon/internal/roles"
	"github.com/rob121/cannon/internal/sites"
	"gorm.io/gorm"
)

var ErrPermissionDenied = errors.New("permission denied")

func hasAnyRole(ctx context.Context, userID uint, names ...string) (bool, error) {
	for _, name := range names {
		ok, err := roles.HasRole(ctx, userID, name)
		if err != nil {
			return false, err
		}
		if ok {
			return true, nil
		}
	}
	return false, nil
}

// CanCreateItem reports whether the user may create content items.
func CanCreateItem(ctx context.Context, userID uint) (bool, error) {
	return hasAnyRole(ctx, userID, roles.AdminRole, roles.EditorRole, roles.AuthorRole, roles.WriterRole)
}

// CanCreateItemInCategory reports whether the user may create items in the given category.
func CanCreateItemInCategory(ctx context.Context, userID uint, categoryID *uint) (bool, error) {
	ok, err := CanCreateItem(ctx, userID)
	if err != nil || !ok {
		return ok, err
	}
	if categoryID == nil || *categoryID == 0 {
		return true, nil
	}
	db, err := sites.DB(ctx)
	if err != nil {
		return false, err
	}
	var cat models.Category
	if err := db.Preload("CreateGroups").First(&cat, *categoryID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return true, nil
		}
		return false, err
	}
	return groups.UserInAnyGroup(ctx, userID, cat.CreateGroups)
}

// CanEditItem reports whether the user may edit the given item.
func CanEditItem(ctx context.Context, userID uint, item *models.Item) (bool, error) {
	ok, err := hasAnyRole(ctx, userID, roles.AdminRole, roles.EditorRole, roles.ManagerRole)
	if err != nil {
		return false, err
	}
	if ok {
		return categoryAllowsEdit(ctx, userID, item)
	}
	author, err := roles.HasRole(ctx, userID, roles.AuthorRole)
	if err != nil {
		return false, err
	}
	writer, err := roles.HasRole(ctx, userID, roles.WriterRole)
	if err != nil {
		return false, err
	}
	if (!author && !writer) || item == nil || item.AuthorID == nil {
		return false, nil
	}
	if *item.AuthorID != userID {
		return false, nil
	}
	return categoryAllowsEdit(ctx, userID, item)
}

func categoryAllowsEdit(ctx context.Context, userID uint, item *models.Item) (bool, error) {
	if item == nil || item.CategoryID == nil || *item.CategoryID == 0 {
		return true, nil
	}
	db, err := sites.DB(ctx)
	if err != nil {
		return false, err
	}
	var cat models.Category
	if err := db.Preload("EditGroups").First(&cat, *item.CategoryID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return true, nil
		}
		return false, err
	}
	return groups.UserInAnyGroup(ctx, userID, cat.EditGroups)
}

// CanPublishItem reports whether the user may publish or unpublish items.
func CanPublishItem(ctx context.Context, userID uint) (bool, error) {
	return hasAnyRole(ctx, userID, roles.AdminRole, roles.EditorRole)
}

// CanDeleteItem reports whether the user may permanently delete items.
func CanDeleteItem(ctx context.Context, userID uint) (bool, error) {
	return hasAnyRole(ctx, userID, roles.AdminRole, roles.EditorRole)
}

// CanManageComments reports whether the user may moderate comments in admin.
func CanManageComments(ctx context.Context, userID uint) (bool, error) {
	return hasAnyRole(ctx, userID, roles.AdminRole, roles.EditorRole)
}
