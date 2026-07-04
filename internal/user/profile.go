package user

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/rob121/cannon/internal/models"
	"github.com/rob121/cannon/internal/sites"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

var (
	ErrUsernameTaken   = errors.New("username is already in use")
	ErrEmailTaken      = errors.New("email is already in use")
	ErrInvalidPassword = errors.New("current password is incorrect")
	ErrPasswordShort   = errors.New("password must be at least 8 characters")
)

// HasLocalPassword reports whether the user can sign in with a password.
func HasLocalPassword(db *gorm.DB, u *models.User) bool {
	if u == nil || strings.TrimSpace(u.Hash) == "" {
		return false
	}
	if u.AuthID == nil {
		return true
	}
	var auth models.Authenticator
	if err := db.First(&auth, *u.AuthID).Error; err != nil {
		return true
	}
	return auth.Name == "local"
}

// IsUsernameAvailable reports whether username is unused or owned by excludeUserID.
func IsUsernameAvailable(db *gorm.DB, username string, excludeUserID uint) (bool, error) {
	username = strings.TrimSpace(username)
	if username == "" {
		return false, fmt.Errorf("username is required")
	}
	var existing models.User
	err := db.Where("username = ?", username).First(&existing).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return true, nil
	}
	if err != nil {
		return false, err
	}
	return existing.UserID == excludeUserID, nil
}

// IsEmailAvailable reports whether email is unused or owned by excludeUserID.
func IsEmailAvailable(db *gorm.DB, email string, excludeUserID uint) (bool, error) {
	email = strings.TrimSpace(email)
	if email == "" {
		return true, nil
	}
	var existing models.User
	err := db.Where("email = ?", email).First(&existing).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return true, nil
	}
	if err != nil {
		return false, err
	}
	return existing.UserID == excludeUserID, nil
}

// UpdateProfileIdentity updates editable account identity fields.
func UpdateProfileIdentity(ctx context.Context, userID uint, username, email string) error {
	db, err := sites.DB(ctx)
	if err != nil {
		return err
	}
	username = strings.TrimSpace(username)
	email = strings.TrimSpace(email)
	if username == "" {
		return fmt.Errorf("username is required")
	}
	ok, err := IsUsernameAvailable(db, username, userID)
	if err != nil {
		return err
	}
	if !ok {
		return ErrUsernameTaken
	}
	ok, err = IsEmailAvailable(db, email, userID)
	if err != nil {
		return err
	}
	if !ok {
		return ErrEmailTaken
	}
	return db.Model(&models.User{}).Where("user_id = ?", userID).Updates(map[string]any{
		"username": username,
		"email":    email,
	}).Error
}

// UpdatePassword changes the user's password after verifying the current one when set.
func UpdatePassword(ctx context.Context, userID uint, currentPassword, newPassword string) error {
	newPassword = strings.TrimSpace(newPassword)
	if len(newPassword) < 8 {
		return ErrPasswordShort
	}
	db, err := sites.DB(ctx)
	if err != nil {
		return err
	}
	var u models.User
	if err := db.First(&u, userID).Error; err != nil {
		return err
	}
	if HasLocalPassword(db, &u) {
		if err := bcrypt.CompareHashAndPassword([]byte(u.Hash), []byte(currentPassword)); err != nil {
			return ErrInvalidPassword
		}
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	updates := map[string]any{"hash": string(hash)}
	if u.AuthID == nil {
		var auth models.Authenticator
		if err := db.Where("name = ?", "local").First(&auth).Error; err == nil {
			updates["auth_id"] = auth.AuthID
		}
	}
	return db.Model(&u).Updates(updates).Error
}
