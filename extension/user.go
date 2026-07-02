package extension

import (
	"fmt"
	"strings"
)

// UserAuthenticated reports whether Cannon attached a signed-in user to the request.
func UserAuthenticated(req WireRequest) bool {
	v, ok := req.User["authenticated"].(bool)
	return ok && v
}

// UserString returns a string field from the user scope (for example "email", "username").
func UserString(req WireRequest, key string) string {
	v, ok := req.User[key]
	if !ok || v == nil {
		return ""
	}
	switch s := v.(type) {
	case string:
		return s
	default:
		return fmt.Sprint(v)
	}
}

// UserID returns the Cannon user id when authenticated.
func UserID(req WireRequest) (uint64, bool) {
	if !UserAuthenticated(req) {
		return 0, false
	}
	v, ok := req.User["user_id"]
	if !ok || v == nil {
		return 0, false
	}
	switch id := v.(type) {
	case float64:
		return uint64(id), true
	case int:
		return uint64(id), true
	case int64:
		return uint64(id), true
	case uint64:
		return id, true
	case uint:
		return uint64(id), true
	default:
		return 0, false
	}
}

// UserDisplayName returns a friendly name for the signed-in user.
// It prefers "given_name family_name", then username, then email.
func UserDisplayName(req WireRequest) string {
	if !UserAuthenticated(req) {
		return ""
	}
	given := strings.TrimSpace(UserString(req, "given_name"))
	family := strings.TrimSpace(UserString(req, "family_name"))
	if given != "" || family != "" {
		return strings.TrimSpace(given + " " + family)
	}
	if username := UserString(req, "username"); username != "" {
		return username
	}
	return UserString(req, "email")
}
