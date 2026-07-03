package content

import (
	"context"
	"errors"

	"github.com/rob121/cannon/internal/groups"
	"github.com/rob121/cannon/internal/models"
	"github.com/rob121/cannon/internal/roles"
)

var ErrPermissionDenied = errors.New("permission denied")

// CanCreateItem reports whether the user may create content items.
func CanCreateItem(ctx context.Context, userID uint) (bool, error) {
	return CanCreateItemInCategory(ctx, userID, nil)
}

// CanCreateItemInCategory reports whether the user may create items in the given category.
func CanCreateItemInCategory(ctx context.Context, userID uint, categoryID *uint) (bool, error) {
	allowed, err := EffectiveCreateGroups(ctx, categoryID)
	if err != nil {
		return false, err
	}
	if len(allowed) > 0 {
		return groups.UserInAnyGroup(ctx, userID, allowed)
	}
	return hasAnyRole(ctx, userID, roles.AdminRole, roles.EditorRole, roles.AuthorRole, roles.WriterRole)
}

// CanEditItem reports whether the user may edit the given item.
func CanEditItem(ctx context.Context, userID uint, item *models.Item) (bool, error) {
	var categoryID *uint
	if item != nil {
		categoryID = item.CategoryID
	}
	allowed, err := EffectiveEditGroups(ctx, categoryID)
	if err != nil {
		return false, err
	}
	if len(allowed) > 0 {
		return groups.UserInAnyGroup(ctx, userID, allowed)
	}
	ok, err := hasAnyRole(ctx, userID, roles.AdminRole, roles.EditorRole, roles.ManagerRole)
	if err != nil {
		return false, err
	}
	if ok {
		return true, nil
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
	return *item.AuthorID == userID, nil
}

// CanPublishItem reports whether the user may publish or unpublish items.
func CanPublishItem(ctx context.Context, userID uint, categoryID *uint) (bool, error) {
	allowed, err := EffectivePublishGroups(ctx, categoryID)
	if err != nil {
		return false, err
	}
	if len(allowed) > 0 {
		return groups.UserInAnyGroup(ctx, userID, allowed)
	}
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
