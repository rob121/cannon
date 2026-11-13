package extension

import (
	"net/http"
	"testing"
)

func TestEncodeDecodeProxyUser(t *testing.T) {
	user := map[string]any{
		"authenticated": true,
		"user_id":       float64(7),
		"permissions":   []any{"cannon-ext-ssh.terminal.use"},
	}
	enc, err := EncodeProxyUser(user)
	if err != nil {
		t.Fatal(err)
	}
	h := http.Header{}
	h.Set(WireUserHeader, enc)
	got, ok := UserFromProxyHeader(h)
	if !ok {
		t.Fatal("expected decoded user")
	}
	if got["user_id"].(float64) != 7 {
		t.Fatalf("user_id: %#v", got["user_id"])
	}
}
