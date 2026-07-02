package templateengine

import (
	"strings"
	"testing"

	"github.com/rob121/cannon/internal/themes"
)

func TestDefaultMailTemplateResponsive(t *testing.T) {
	e := New(nil, themes.Selection{}, nil, nil, nil)
	tmpl, err := e.parse("default/mail/default.html")
	if err != nil {
		t.Fatal(err)
	}
	var buf strings.Builder
	err = tmpl.Execute(&buf, map[string]any{
		"Subject":       "Reset your password",
		"SiteName":      "Cannon Demo",
		"Body":          "Use the button below to reset your password.",
		"ActionURL":     "https://example.com/account/reset?token=abc",
		"ActionLabel":   "Reset password",
		"ActionURLText": "https://example.com/account/reset?token=abc",
	})
	if err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	for _, part := range []string{
		`<!DOCTYPE html>`,
		`role="presentation"`,
		`max-width: 600px`,
		`@media all and (max-width: 639px)`,
		`format-detection`,
		`x-apple-disable-message-reformatting`,
		`Reset your password`,
		`Cannon Demo`,
		`bgcolor="#059669"`,
		`btn-full`,
	} {
		if !strings.Contains(out, part) {
			t.Fatalf("missing %q in mail template output", part)
		}
	}
}
