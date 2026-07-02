package content

import (
	"context"
	"errors"

	"github.com/rob121/cannon/internal/models"
	"github.com/rob121/cannon/internal/roles"
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

// CanEditItem reports whether the user may edit the given item.
func CanEditItem(ctx context.Context, userID uint, item *models.Item) (bool, error) {
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
