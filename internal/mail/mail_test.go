package mail_test

import (
	"strings"
	"testing"

	"github.com/rob121/cannon/internal/mail"
)

func TestConfiguredRequiresHostAndFrom(t *testing.T) {
	cfg := mail.Settings{Host: "smtp.example.com"}
	if cfg.Configured() {
		t.Fatal("expected unconfigured without from email")
	}
	cfg.FromEmail = "noreply@example.com"
	if !cfg.Configured() {
		t.Fatal("expected configured")
	}
}

func TestSendRequiresRecipient(t *testing.T) {
	cfg := mail.Settings{Host: "smtp.example.com", FromEmail: "noreply@example.com"}
	err := mail.Send(t.Context(), cfg, mail.Message{Subject: "Hi", Text: "Hello"})
	if err == nil || !strings.Contains(err.Error(), "recipient") {
		t.Fatalf("expected recipient error, got %v", err)
	}
}

func TestHTMLTemplatePathDefault(t *testing.T) {
	cfg := mail.Settings{}
	if got := cfg.HTMLTemplatePath(); got != "default/mail/default.html" {
		t.Fatalf("got %q", got)
	}
}
