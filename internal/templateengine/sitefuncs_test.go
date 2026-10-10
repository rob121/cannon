package templateengine

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/rob121/cannon/internal/models"
)

func TestUserNameFromModel(t *testing.T) {
	got := UserName(models.User{GivenName: "Jane", FamilyName: "Doe", Username: "jdoe"})
	if got != "Jane Doe" {
		t.Fatalf("UserName() = %q", got)
	}
	got = UserName(models.User{Username: "jdoe"})
	if got != "jdoe" {
		t.Fatalf("UserName(username only) = %q", got)
	}
}

func TestUserNameOverride(t *testing.T) {
	user := models.User{GivenName: "Jane", FamilyName: "Doe", Username: "jdoe"}
	got := UserName(user, "Display Name")
	if got != "Display Name" {
		t.Fatalf("UserName(override) = %q", got)
	}
}

func TestItemDateUsesPublishStart(t *testing.T) {
	published := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
	item := models.Item{
		PublishStart: &published,
		CreatedAt:    time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
	}
	got := ItemDate(item, "2006-01-02")
	if got != "2026-03-01" {
		t.Fatalf("ItemDate() = %q", got)
	}
}

func TestExcerptStripsHTML(t *testing.T) {
	got := Excerpt("<p>Hello <strong>world</strong></p>", 20)
	if got != "Hello world" {
		t.Fatalf("Excerpt() = %q", got)
	}
}

func TestThemeAssetURL(t *testing.T) {
	if got := ThemeAssetURL("site.css"); got != "/theme/site.css" {
		t.Fatalf("ThemeAssetURL() = %q", got)
	}
}

func TestIsCurrentPath(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/blog/post", nil)
	if !IsCurrentPath(r, "/blog/post") {
		t.Fatal("expected exact path match")
	}
	if !IsCurrentPath(r, "/blog") {
		t.Fatal("expected prefix path match")
	}
	if IsCurrentPath(r, "/other") {
		t.Fatal("expected different path to not match")
	}
}

func TestAbsoluteURL(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/hello", nil)
	r.Host = "example.test"
	got := AbsoluteURL(r, "/path")
	if got != "http://example.test/path" {
		t.Fatalf("AbsoluteURL() = %q", got)
	}
}

func TestPublicSettingKeyAllowlist(t *testing.T) {
	if !PublicSettingKey("general", "site_name") {
		t.Fatal("expected site_name to be public")
	}
	if PublicSettingKey("mail", "smtp_host") {
		t.Fatal("expected mail settings to be private")
	}
}
