package auth

import (
	"github.com/rob121/cannon/internal/controllers"
	"github.com/rob121/cannon/internal/hooks"
	"github.com/rob121/cannon/internal/models"
	"github.com/rob121/cannon/internal/sites"
	"github.com/rob121/cannon/internal/user"
)

func loadUserByID(ctx *controllers.Context, userID uint) (*models.User, error) {
	db, err := sites.DB(ctx.GoContext())
	if err != nil {
		return nil, err
	}
	var u models.User
	if err := db.First(&u, userID).Error; err != nil {
		return nil, err
	}
	return &u, nil
}

func completeFrontendLogin(ctx *controllers.Context, userID uint, loginContext string) error {
	if err := ctx.User.Login(userID); err != nil {
		return err
	}
	_ = user.EnsureRegisteredGroup(ctx.GoContext(), userID)
	u, err := loadUserByID(ctx, userID)
	if err != nil {
		return err
	}
	afterArgs := map[string]any{
		"context":  loginContext,
		"user_id":  u.UserID,
		"username": u.Username,
		"email":    u.Email,
	}
	_, err = hooks.Fire(ctx.GoContext(), hooks.OnUserAfterLogin, afterArgs)
	return err
}
