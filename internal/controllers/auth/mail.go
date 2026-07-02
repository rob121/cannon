package auth

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/rob121/cannon/internal/controllers"
	"github.com/rob121/cannon/internal/httpx"
	"github.com/rob121/cannon/internal/mail"
	"github.com/rob121/cannon/internal/models"
	"github.com/rob121/cannon/internal/settings"
)

func sendPasswordResetEmail(ctx *controllers.Context, u models.User, token string) {
	if strings.TrimSpace(u.Email) == "" {
		return
	}
	cfg, err := mail.LoadSettings(ctx.GoContext())
	if err != nil || !cfg.Configured() {
		return
	}
	resetPath := ResetURL(ctx.GoContext(), token)
	resetURL := httpx.AbsoluteURL(ctx.Request, resetPath)
	subject := "Reset your password"
	text := fmt.Sprintf("Use this link to reset your password:\n\n%s\n", resetURL)
	msg := mail.Message{
		To:      u.Email,
		Subject: subject,
		Text:    text,
	}
	if cfg.UseHTML {
		siteName, _ := settings.GlobalString(ctx.GoContext(), settings.SectionGeneral, "site_name")
		data := map[string]any{
			"Subject":       subject,
			"Body":          "Use the button below to reset your password.",
			"ActionURL":     resetURL,
			"ActionLabel":   "Reset password",
			"ActionURLText": resetURL,
			"SiteName":      siteName,
		}
		if ctx.Template != nil {
			var buf bytes.Buffer
			if err := ctx.Template.RenderFragment(&buf, cfg.HTMLTemplatePath(), data); err == nil {
				msg.HTML = buf.String()
			}
		}
	}
	_ = mail.Send(ctx.GoContext(), cfg, msg)
}
