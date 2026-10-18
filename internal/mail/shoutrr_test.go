package mail_test

import (
	"net/url"
	"testing"

	"github.com/rob121/cannon/internal/mail"
)

func TestShoutrrSMTPURL(t *testing.T) {
	cfg := mail.Settings{
		Host:      "smtp.example.com",
		Port:      587,
		Username:  "user",
		Password:  "p@ss:word",
		FromEmail: "noreply@example.com",
	}
	got, err := mail.ShoutrrSMTPURL(cfg, "alerts@example.com")
	if err != nil {
		t.Fatal(err)
	}
	u, err := url.Parse(got)
	if err != nil {
		t.Fatal(err)
	}
	if u.Scheme != "smtp" || u.Hostname() != "smtp.example.com" || u.Port() != "587" {
		t.Fatalf("url host = %s", got)
	}
	if u.User.Username() != "user" {
		t.Fatalf("user = %q", u.User.Username())
	}
	pass, _ := u.User.Password()
	if pass != "p@ss:word" {
		t.Fatalf("pass = %q", pass)
	}
	if u.Query().Get("from") != "noreply@example.com" || u.Query().Get("to") != "alerts@example.com" {
		t.Fatalf("query = %v", u.Query())
	}
	to, ok := mail.SiteMailRecipient(cfg, got)
	if !ok || to != "alerts@example.com" {
		t.Fatalf("SiteMailRecipient = %q %v", to, ok)
	}
}

func TestShoutrrSMTPURLRequiresConfig(t *testing.T) {
	if _, err := mail.ShoutrrSMTPURL(mail.Settings{}, "a@b.com"); err == nil {
		t.Fatal("expected error")
	}
	cfg := mail.Settings{Host: "smtp.example.com", FromEmail: "a@b.com"}
	if _, err := mail.ShoutrrSMTPURL(cfg, ""); err == nil {
		t.Fatal("expected recipient error")
	}
}
