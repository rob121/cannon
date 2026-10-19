package notifications

import (
	"testing"
)

func TestFormatEmailSubmission(t *testing.T) {
	subject, body := formatEmail("onSubmissionAfterCreate", map[string]any{
		"title":             "Road closure",
		"submission_id":     7,
		"contributor_name":  "Jane",
		"contributor_email": "jane@example.com",
		"admin_url":         "/admin/extension-apps/cannon-ext-submission/submissions/7",
	})
	if subject != "New submission: Road closure" {
		t.Fatalf("subject = %q", subject)
	}
	for _, want := range []string{
		"Event: onSubmissionAfterCreate",
		"title: Road closure",
		"submission_id: 7",
		"contributor_email: jane@example.com",
		"admin_url: /admin/extension-apps/cannon-ext-submission/submissions/7",
	} {
		if !containsLine(body, want) {
			t.Fatalf("body missing %q:\n%s", want, body)
		}
	}
}

func containsLine(body, want string) bool {
	for _, line := range splitLines(body) {
		if line == want {
			return true
		}
	}
	return false
}

func splitLines(s string) []string {
	var out []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			out = append(out, s[start:i])
			start = i + 1
		}
	}
	out = append(out, s[start:])
	return out
}
