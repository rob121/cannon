package mail

import (
	"context"
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"net"
	"net/smtp"
	"strings"

	"github.com/rob121/cannon/internal/settings"
)

const SettingsSection = settings.SectionMail

const defaultHTMLTemplate = "default/mail/default.html"

// Settings holds global SMTP configuration.
type Settings struct {
	Host         string
	Port         int
	Username     string
	Password     string
	FromEmail    string
	FromName     string
	UseTLS       bool
	UseHTML      bool
	HTMLTemplate string
}

// Message is an outgoing email.
type Message struct {
	To      string
	Subject string
	Text    string
	HTML    string
}

// LoadSettings reads mail settings from the global settings store.
func LoadSettings(ctx context.Context) (Settings, error) {
	data, err := settings.NewStore().Load(ctx, settings.ScopeGlobal, SettingsSection)
	if err != nil {
		return Settings{}, err
	}
	port, err := settingsInt(data, "smtp_port", 587)
	if err != nil {
		return Settings{}, err
	}
	cfg := Settings{
		Host:         settingsString(data, "smtp_host", ""),
		Port:         port,
		Username:     settingsString(data, "smtp_username", ""),
		Password:     settingsString(data, "smtp_password", ""),
		FromEmail:    settingsString(data, "from_email", ""),
		FromName:     settingsString(data, "from_name", ""),
		UseTLS:       settingsBool(data, "use_tls", true),
		UseHTML:      settingsBool(data, "use_html", false),
		HTMLTemplate: settingsString(data, "html_template", defaultHTMLTemplate),
	}
	if cfg.HTMLTemplate == "" {
		cfg.HTMLTemplate = defaultHTMLTemplate
	}
	return cfg, nil
}

// Configured reports whether SMTP is ready to send mail.
func (cfg Settings) Configured() bool {
	return strings.TrimSpace(cfg.Host) != "" && strings.TrimSpace(cfg.FromEmail) != ""
}

// HTMLTemplatePath returns the configured HTML template path.
func (cfg Settings) HTMLTemplatePath() string {
	if strings.TrimSpace(cfg.HTMLTemplate) == "" {
		return defaultHTMLTemplate
	}
	return strings.TrimSpace(cfg.HTMLTemplate)
}

// Send delivers a message using global SMTP settings.
func Send(ctx context.Context, cfg Settings, msg Message) error {
	if !cfg.Configured() {
		return fmt.Errorf("mail is not configured")
	}
	to := strings.TrimSpace(msg.To)
	if to == "" {
		return fmt.Errorf("recipient email is required")
	}
	subject := strings.TrimSpace(msg.Subject)
	if subject == "" {
		subject = "Message from Cannon"
	}
	text := strings.TrimSpace(msg.Text)
	html := strings.TrimSpace(msg.HTML)
	if text == "" && html == "" {
		return fmt.Errorf("message body is required")
	}
	if html == "" && cfg.UseHTML && text != "" {
		html = wrapTextAsHTML(text)
	}

	from := formatAddress(cfg.FromName, cfg.FromEmail)
	var body []byte
	if html != "" {
		body = buildMultipartBody(from, to, subject, text, html)
	} else {
		body = buildTextBody(from, to, subject, text)
	}

	addr := net.JoinHostPort(cfg.Host, fmt.Sprintf("%d", cfg.Port))
	auth := smtpAuth(cfg)
	if cfg.UseTLS {
		return sendMailTLS(addr, auth, cfg.FromEmail, []string{to}, body)
	}
	return smtp.SendMail(addr, auth, cfg.FromEmail, []string{to}, body)
}

func smtpAuth(cfg Settings) smtp.Auth {
	if strings.TrimSpace(cfg.Username) == "" {
		return nil
	}
	return smtp.PlainAuth("", cfg.Username, cfg.Password, cfg.Host)
}

func sendMailTLS(addr string, auth smtp.Auth, from string, to []string, body []byte) error {
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		return err
	}
	conn, err := tls.Dial("tcp", addr, &tls.Config{ServerName: host})
	if err != nil {
		return err
	}
	client, err := smtp.NewClient(conn, host)
	if err != nil {
		return err
	}
	defer client.Close()
	if auth != nil {
		if err := client.Auth(auth); err != nil {
			return err
		}
	}
	if err := client.Mail(from); err != nil {
		return err
	}
	for _, rcpt := range to {
		if err := client.Rcpt(rcpt); err != nil {
			return err
		}
	}
	w, err := client.Data()
	if err != nil {
		return err
	}
	if _, err := w.Write(body); err != nil {
		return err
	}
	if err := w.Close(); err != nil {
		return err
	}
	return client.Quit()
}

func formatAddress(name, email string) string {
	email = strings.TrimSpace(email)
	name = strings.TrimSpace(name)
	if name == "" {
		return email
	}
	return fmt.Sprintf("%s <%s>", encodeHeaderWord(name), email)
}

func encodeHeaderWord(value string) string {
	if value == "" {
		return ""
	}
	for _, r := range value {
		if r > 127 {
			return fmt.Sprintf("=?UTF-8?B?%s?=", base64.StdEncoding.EncodeToString([]byte(value)))
		}
	}
	return value
}

func buildTextBody(from, to, subject, text string) []byte {
	if text == "" {
		text = " "
	}
	return []byte(strings.Join([]string{
		"From: " + from,
		"To: " + to,
		"Subject: " + encodeHeaderWord(subject),
		"MIME-Version: 1.0",
		"Content-Type: text/plain; charset=UTF-8",
		"Content-Transfer-Encoding: 8bit",
		"",
		text,
	}, "\r\n"))
}

func buildMultipartBody(from, to, subject, text, html string) []byte {
	if text == "" {
		text = stripHTML(html)
	}
	boundary := "cannon-mail-boundary"
	return []byte(strings.Join([]string{
		"From: " + from,
		"To: " + to,
		"Subject: " + encodeHeaderWord(subject),
		"MIME-Version: 1.0",
		"Content-Type: multipart/alternative; boundary=" + boundary,
		"",
		"--" + boundary,
		"Content-Type: text/plain; charset=UTF-8",
		"Content-Transfer-Encoding: 8bit",
		"",
		text,
		"--" + boundary,
		"Content-Type: text/html; charset=UTF-8",
		"Content-Transfer-Encoding: 8bit",
		"",
		html,
		"--" + boundary + "--",
		"",
	}, "\r\n"))
}

func wrapTextAsHTML(text string) string {
	escaped := strings.ReplaceAll(text, "&", "&amp;")
	escaped = strings.ReplaceAll(escaped, "<", "&lt;")
	escaped = strings.ReplaceAll(escaped, ">", "&gt;")
	escaped = strings.ReplaceAll(escaped, "\n", "<br>")
	return "<html><body><p>" + escaped + "</p></body></html>"
}

func stripHTML(html string) string {
	replacer := strings.NewReplacer("<br>", "\n", "<br/>", "\n", "<br />", "\n", "<p>", "", "</p>", "\n")
	out := replacer.Replace(html)
	out = strings.ReplaceAll(out, "<", "")
	out = strings.ReplaceAll(out, ">", "")
	return strings.TrimSpace(out)
}

func settingsString(data map[string]any, key, def string) string {
	v, ok := data[key]
	if !ok || v == nil {
		return def
	}
	if s, ok := v.(string); ok {
		return strings.TrimSpace(s)
	}
	return strings.TrimSpace(fmt.Sprint(v))
}

func settingsBool(data map[string]any, key string, def bool) bool {
	v, ok := data[key]
	if !ok || v == nil {
		return def
	}
	switch b := v.(type) {
	case bool:
		return b
	case float64:
		return b != 0
	case string:
		return b == "true" || b == "1"
	default:
		return def
	}
}

func settingsInt(data map[string]any, key string, def int) (int, error) {
	v, ok := data[key]
	if !ok || v == nil {
		return def, nil
	}
	switch n := v.(type) {
	case float64:
		return int(n), nil
	case int:
		return n, nil
	case int64:
		return int(n), nil
	default:
		return def, nil
	}
}
