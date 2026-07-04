package api

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	ctrlauth "github.com/rob121/cannon/internal/controllers/auth"
	"github.com/rob121/cannon/internal/httpx"
	"github.com/rob121/cannon/internal/mail"
	"github.com/rob121/cannon/internal/models"
	"github.com/rob121/cannon/internal/sites"
	"golang.org/x/crypto/bcrypt"
)

func sendPasswordResetEmail(r *http.Request, u models.User, token string) {
	if strings.TrimSpace(u.Email) == "" {
		return
	}
	ctx := r.Context()
	cfg, err := mail.LoadSettings(ctx)
	if err != nil || !cfg.Configured() {
		return
	}
	resetURL := httpx.AbsoluteURL(r, ctrlauth.ResetURL(ctx, token))
	subject := "Reset your password"
	msg := mail.Message{
		To:      u.Email,
		Subject: subject,
		Text:    fmt.Sprintf("Use this link to reset your password:\n\n%s\n", resetURL),
	}
	_ = mail.Send(ctx, cfg, msg)
}

func sendVerificationEmail(r *http.Request, u models.User, token string) {
	if strings.TrimSpace(u.Email) == "" {
		return
	}
	ctx := r.Context()
	cfg, err := mail.LoadSettings(ctx)
	if err != nil || !cfg.Configured() {
		return
	}
	verifyURL := httpx.AbsoluteURL(r, ctrlauth.VerifyURL(ctx, token))
	subject := "Verify your account"
	msg := mail.Message{
		To:      u.Email,
		Subject: subject,
		Text:    fmt.Sprintf("Use this link to verify your account:\n\n%s\n", verifyURL),
	}
	_ = mail.Send(ctx, cfg, msg)
}

func setUserPasswordHash(ctx context.Context, userID uint, password string) error {
	if len(password) < 8 {
		return fmt.Errorf("password must be at least 8 characters")
	}
	db, err := sites.DB(ctx)
	if err != nil {
		return err
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	return db.Model(&models.User{}).Where("user_id = ?", userID).Update("hash", string(hash)).Error
}
