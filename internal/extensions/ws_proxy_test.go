package extensions

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/rob121/cannon/extension"
)

func TestEncodeProxyUserRoundTrip(t *testing.T) {
	user := map[string]any{"authenticated": true, "permissions": []any{"cannon-ext-ssh.terminal.use"}}
	enc, err := extension.EncodeProxyUser(user)
	if err != nil {
		t.Fatal(err)
	}
	h := http.Header{}
	h.Set(extension.WireUserHeader, enc)
	got, ok := extension.UserFromProxyHeader(h)
	if !ok || got["authenticated"] != true {
		t.Fatalf("user: %#v ok=%v", got, ok)
	}
}

func TestAdminWebSocketSuffix(t *testing.T) {
	if AdminWebSocketSuffix() != "/ws" {
		t.Fatalf("suffix: %q", AdminWebSocketSuffix())
	}
}

func TestProxyWebSocketBridge(t *testing.T) {
	socketPath, _ := startUnixHTTPServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, ok := extension.UserFromProxyHeader(r.Header); !ok {
			http.Error(w, "user required", http.StatusForbidden)
			return
		}
		up := websocket.Upgrader{CheckOrigin: func(*http.Request) bool { return true }}
		c, err := up.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer c.Close()
		_ = c.WriteMessage(websocket.TextMessage, []byte("ok"))
		for {
			if _, _, err := c.ReadMessage(); err != nil {
				return
			}
		}
	}))

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user := map[string]any{"authenticated": true, "permissions": []any{"cannon-ext-ssh.terminal.use"}}
		if err := proxyWebSocketToSocket(r.Context(), socketPath, DefaultExtensionTerminalWSPath, w, r, user); err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
		}
	}))
	defer ts.Close()

	wsURL := "ws" + ts.URL[4:]
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	_ = conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	_, msg, err := conn.ReadMessage()
	if err != nil {
		t.Fatal(err)
	}
	if string(msg) != "ok" {
		t.Fatalf("msg %q", msg)
	}
	_ = context.Background()
	_ = net.Dial
}
