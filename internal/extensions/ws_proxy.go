package extensions

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gorilla/websocket"
	"github.com/rob121/cannon/extension"
)

const extensionAdminWebSocketSuffix = "/ws"

// DefaultExtensionTerminalWSPath is the extension unix-socket path proxied for admin terminals.
const DefaultExtensionTerminalWSPath = "/terminal/ws"

var clientWSUpgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

// ProxyAdminWebSocket terminates the browser WebSocket and opens a second socket to the extension.
func (m *Manager) ProxyAdminWebSocket(ctx context.Context, extensionName, backendPath string, w http.ResponseWriter, r *http.Request, userCtx map[string]any) error {
	rt, err := m.ensureActiveRuntime(ctx, extensionName)
	if err != nil {
		return err
	}
	return proxyWebSocketToSocket(ctx, rt.Model.Socket, backendPath, w, r, userCtx)
}

func proxyWebSocketToSocket(ctx context.Context, socketPath, backendPath string, w http.ResponseWriter, r *http.Request, userCtx map[string]any) error {
	if backendPath == "" {
		backendPath = DefaultExtensionTerminalWSPath
	}
	if !strings.HasPrefix(backendPath, "/") {
		backendPath = "/" + backendPath
	}

	userEncoded, err := extension.EncodeProxyUser(userCtx)
	if err != nil {
		return err
	}

	backendURL := url.URL{Scheme: "ws", Host: "cannon.local", Path: backendPath}
	dialer := websocket.Dialer{
		HandshakeTimeout: 10 * time.Second,
		NetDialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			var d net.Dialer
			return d.DialContext(ctx, "unix", socketPath)
		},
	}
	hdr := http.Header{}
	hdr.Set(extension.WireUserHeader, userEncoded)
	backendConn, backendResp, err := dialer.DialContext(ctx, backendURL.String(), hdr)
	if err != nil {
		if backendResp != nil && backendResp.Body != nil {
			_ = backendResp.Body.Close()
		}
		if backendResp != nil && backendResp.StatusCode == http.StatusForbidden {
			http.Error(w, "forbidden", http.StatusForbidden)
			return fmt.Errorf("extension websocket forbidden")
		}
		http.Error(w, "extension websocket unavailable", http.StatusBadGateway)
		return fmt.Errorf("dial extension websocket: %w", err)
	}

	clientConn, err := clientWSUpgrader.Upgrade(w, r, nil)
	if err != nil {
		_ = backendConn.Close()
		return err
	}

	go bridgeWebSockets(clientConn, backendConn)
	return nil
}

func bridgeWebSockets(client, backend *websocket.Conn) {
	defer client.Close()
	defer backend.Close()

	errc := make(chan error, 2)
	copyWS := func(dst, src *websocket.Conn) {
		for {
			msgType, msg, err := src.ReadMessage()
			if err != nil {
				errc <- err
				return
			}
			if err := dst.WriteMessage(msgType, msg); err != nil {
				errc <- err
				return
			}
		}
	}
	go copyWS(client, backend)
	go copyWS(backend, client)
	<-errc
}

// AdminWebSocketSuffix reports the URL suffix for extension admin WebSocket proxying.
func AdminWebSocketSuffix() string {
	return extensionAdminWebSocketSuffix
}
