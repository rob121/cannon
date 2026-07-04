package admin

import "testing"

func TestThemeTypeLabel(t *testing.T) {
	if got := themeTypeLabel("frontend"); got != "Frontend" {
		t.Fatalf("got %q", got)
	}
	if got := themeTypeLabel("backend"); got != "Admin" {
		t.Fatalf("got %q", got)
	}
	if got := themeTypeLabel("full"); got != "Full" {
		t.Fatalf("got %q", got)
	}
}

func TestRoleDisplayName(t *testing.T) {
	if got := RoleDisplayName("administrator"); got != "Administrator" {
		t.Fatalf("got %q", got)
	}
	if got := RoleDisplayName("content_editor"); got != "Content Editor" {
		t.Fatalf("got %q", got)
	}
}
