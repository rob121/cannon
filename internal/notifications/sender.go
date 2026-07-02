package notifications

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/smtp"
	"net/url"
	"strings"
	"time"
)

func sendShoutrr(ctx context.Context, rawURL, message string) error {
	u, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("parse shoutrrr url: %w", err)
	}
	switch strings.ToLower(u.Scheme) {
	case "smtp", "smtps":
		return sendSMTP(u, message)
	case "slack":
		return sendSlackWebhook(u, message)
	default:
		return sendGenericWebhook(ctx, rawURL, message)
	}
}

func sendSMTP(u *url.URL, message string) error {
	host := u.Hostname()
	port := u.Port()
	if port == "" {
		port = "587"
	}
	addr := net.JoinHostPort(host, port)
	user := u.User.Username()
	pass, _ := u.User.Password()
	from := u.Query().Get("from")
	if from == "" {
		from = user
	}
	to := u.Query().Get("to")
	if to == "" {
		return fmt.Errorf("smtp url requires to= query parameter")
	}
	subject := u.Query().Get("subject")
	if subject == "" {
		subject = "Cannon Notification"
	}
	body := strings.Join([]string{
		"From: " + from,
		"To: " + to,
		"Subject: " + subject,
		"MIME-Version: 1.0",
		"Content-Type: text/plain; charset=UTF-8",
		"",
		message,
	}, "\r\n")
	auth := smtp.PlainAuth("", user, pass, host)
	return smtp.SendMail(addr, auth, from, []string{to}, []byte(body))
}

func sendSlackWebhook(u *url.URL, message string) error {
	token := u.User.Username()
	if token == "" {
		token, _ = u.User.Password()
	}
	webhookURL := fmt.Sprintf("https://hooks.slack.com/services/%s", token)
	if strings.Contains(token, "/") {
		webhookURL = "https://hooks.slack.com/services/" + strings.TrimPrefix(token, "/")
	}
	req, err := http.NewRequest(http.MethodPost, webhookURL, strings.NewReader(`{"text":`+quoteJSON(message)+`}`))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return fmt.Errorf("slack webhook status %s", resp.Status)
	}
	return nil
}

func sendGenericWebhook(ctx context.Context, rawURL, message string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, rawURL, strings.NewReader(message))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "text/plain; charset=utf-8")
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return fmt.Errorf("webhook status %s", resp.Status)
	}
	return nil
}

func quoteJSON(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `"`, `\"`)
	return `"` + s + `"`
}
