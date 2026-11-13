package extension

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"strings"
)

// WireUserHeader is set by Cannon when proxying authenticated admin WebSocket upgrades.
const WireUserHeader = "X-Cannon-Wire-User"

// EncodeProxyUser serializes the Cannon user scope for WebSocket proxy requests.
func EncodeProxyUser(user map[string]any) (string, error) {
	if user == nil {
		user = map[string]any{"authenticated": false}
	}
	raw, err := json.Marshal(user)
	if err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(raw), nil
}

// UserFromProxyHeader decodes the user scope from a proxied WebSocket upgrade.
func UserFromProxyHeader(h http.Header) (map[string]any, bool) {
	val := strings.TrimSpace(h.Get(WireUserHeader))
	if val == "" {
		return nil, false
	}
	raw, err := base64.RawURLEncoding.DecodeString(val)
	if err != nil {
		return nil, false
	}
	var user map[string]any
	if err := json.Unmarshal(raw, &user); err != nil {
		return nil, false
	}
	return user, true
}
