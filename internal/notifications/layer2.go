package notifications

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/rob121/cannon/internal/mail"
)

func dispatchLayer2(ctx context.Context, event string, args map[string]any) {
	recipients, err := resolveEmailRecipients(ctx, event)
	if err != nil {
		log.Printf("notifications: resolve recipients for %s: %v", event, err)
		return
	}
	if len(recipients) == 0 {
		return
	}
	cfg, err := mail.LoadSettings(ctx)
	if err != nil {
		log.Printf("notifications: mail settings for %s: %v", event, err)
		return
	}
	if !cfg.Configured() {
		log.Printf("notifications: mail not configured; skipping layer 2 for %s", event)
		return
	}
	subject, body := formatEmail(event, args)
	for _, rcpt := range recipients {
		if err := mail.Send(ctx, cfg, mail.Message{
			To:      rcpt.Email,
			Subject: subject,
			Text:    body,
		}); err != nil {
			log.Printf("notifications: email %s for %s (user %d): %v", rcpt.Email, event, rcpt.UserID, err)
		}
	}
}

func formatEmail(event string, args map[string]any) (subject, body string) {
	subject = "Cannon: " + event
	lines := []string{"Event: " + event}
	for _, key := range []string{"username", "email", "user_id", "item_id", "comment_id"} {
		if v, ok := args[key]; ok && fmt.Sprint(v) != "" {
			lines = append(lines, fmt.Sprintf("%s: %v", key, v))
		}
	}
	body = strings.Join(lines, "\n")
	return subject, body
}
