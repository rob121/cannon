package settings_test

import (
	"testing"

	"github.com/rob121/cannon/internal/settings"
)

func TestNormalizeAdminAccent(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{"", settings.AdminAccentGreen},
		{"green", settings.AdminAccentGreen},
		{"BLUE", settings.AdminAccentBlue},
		{" dark-gray ", settings.AdminAccentDarkGray},
		{"dark_gray", settings.AdminAccentDarkGray},
		{"purple", settings.AdminAccentPurple},
		{"nope", settings.AdminAccentGreen},
	}
	for _, tc := range tests {
		if got := settings.NormalizeAdminAccent(tc.in); got != tc.want {
			t.Fatalf("NormalizeAdminAccent(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}
