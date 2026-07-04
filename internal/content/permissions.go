package content

import (
	"context"
	"errors"

	"github.com/rob121/cannon/internal/models"
	"github.com/rob121/cannon/internal/security"
)

var ErrPermissionDenied = errors.New("permission denied")

// CanCreateItem reports whether the user may create content items.
func CanCreateItem(ctx context.Context, userID uint) (bool, error) {
	return CanCreateItemInCategory(ctx, userID, nil)
}

// CanCreateItemInCategory reports whether the user may create items in the given category.
func CanCreateItemInCategory(ctx context.Context, userID uint, categoryID *uint) (bool, error) {
	return security.Can(ctx, userID, security.PermContentFrontendCreate)
}

// CanEditItem reports whether the user may edit the given item.
func CanEditItem(ctx context.Context, userID uint, item *models.Item) (bool, error) {
	if ok, err := security.Can(ctx, userID, security.PermContentFrontendEdit); err != nil {
		return false, err
	} else if ok {
		return true, nil
	}
	if ok, err := security.Can(ctx, userID, security.PermContentFrontendEditOwn); err != nil {
		return false, err
	} else if ok && item != nil && item.AuthorID != nil && *item.AuthorID == userID {
		return true, nil
	}
	return false, nil
}

// CanPublishItem reports whether the user may publish or unpublish items.
func CanPublishItem(ctx context.Context, userID uint, categoryID *uint) (bool, error) {
	return security.Can(ctx, userID, security.PermContentFrontendPublish)
}

// CanDeleteItem reports whether the user may permanently delete items.
func CanDeleteItem(ctx context.Context, userID uint) (bool, error) {
	return security.Can(ctx, userID, security.PermContentFrontendDelete)
}

// CanViewFrontendContent reports whether the user may access protected frontend content features.
func CanViewFrontendContent(ctx context.Context, userID uint) (bool, error) {
	return security.Can(ctx, userID, security.PermContentFrontendView)
}

// CanViewComments reports whether the user may view item comments on the frontend.
func CanViewComments(ctx context.Context, userID uint) (bool, error) {
	return security.Can(ctx, userID, security.PermContentFrontendCommentView)
}

// CanCreateComment reports whether the user may post comments on the frontend.
func CanCreateComment(ctx context.Context, userID uint) (bool, error) {
	return security.Can(ctx, userID, security.PermContentFrontendCommentCreate)
}

// CanModerateComments reports whether the user may moderate comments.
func CanModerateComments(ctx context.Context, userID uint) (bool, error) {
	return security.Can(ctx, userID, security.PermContentFrontendCommentModerate)
}

// CanManageComments reports whether the user may moderate comments in admin.
func CanManageComments(ctx context.Context, userID uint) (bool, error) {
	return CanModerateComments(ctx, userID)
}
