package content

import (
	"net/http"
	"strings"
	"time"
)

// FormBool reads a checkbox-style boolean from a form.
func FormBool(r *http.Request, key string) bool {
	v := strings.TrimSpace(r.FormValue(key))
	return v == "on" || v == "1" || v == "true" || v == "yes"
}

// FormTimePtr parses a datetime-local or date form value.
func FormTimePtr(r *http.Request, key string) *time.Time {
	raw := strings.TrimSpace(r.FormValue(key))
	if raw == "" {
		return nil
	}
	layouts := []string{
		time.RFC3339,
		"2006-01-02T15:04:05",
		"2006-01-02T15:04",
		"2006-01-02",
	}
	for _, layout := range layouts {
		if t, err := time.Parse(layout, raw); err == nil {
			return &t
		}
	}
	return nil
}
