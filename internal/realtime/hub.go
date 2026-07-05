package realtime

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/centrifugal/centrifuge"
)

// Hub wraps a Centrifuge node for site presence and analytics.
type Hub struct {
	node         *centrifuge.Node
	ws           *centrifuge.WebsocketHandler
	publishMu    sync.Mutex
	publishAfter map[string]*time.Timer
}

// NewHub creates and starts a Centrifuge node.
func NewHub() (*Hub, error) {
	node, err := centrifuge.New(centrifuge.Config{})
	if err != nil {
		return nil, err
	}

	h := &Hub{
		node:         node,
		publishAfter: map[string]*time.Timer{},
	}
	h.configureNode()
	if err := node.Run(); err != nil {
		return nil, err
	}

	h.ws = centrifuge.NewWebsocketHandler(node, centrifuge.WebsocketConfig{
		CheckOrigin: func(r *http.Request) bool {
			// Site middleware already resolved the host; same-origin is enforced by browsers.
			return true
		},
	})
	return h, nil
}

func (h *Hub) configureNode() {
	h.node.OnConnecting(func(ctx context.Context, e centrifuge.ConnectEvent) (centrifuge.ConnectReply, error) {
		cred, ok := centrifuge.GetCredentials(ctx)
		if !ok || cred == nil {
			return centrifuge.ConnectReply{}, centrifuge.ErrorUnauthorized
		}
		return centrifuge.ConnectReply{Credentials: cred}, nil
	})

	h.node.OnConnect(func(client *centrifuge.Client) {
		client.OnSubscribe(func(e centrifuge.SubscribeEvent, cb centrifuge.SubscribeCallback) {
			info, ok := parseConnInfo(client)
			if !ok {
				cb(centrifuge.SubscribeReply{}, centrifuge.ErrorPermissionDenied)
				return
			}
			siteID, kind, valid := ChannelKind(e.Channel)
			if !valid || siteID != info.SiteID {
				cb(centrifuge.SubscribeReply{}, centrifuge.ErrorPermissionDenied)
				return
			}
			switch kind {
			case "analytics":
				if !info.Admin {
					cb(centrifuge.SubscribeReply{}, centrifuge.ErrorPermissionDenied)
					return
				}
				cb(centrifuge.SubscribeReply{
					Options: centrifuge.SubscribeOptions{
						EnableRecovery: true,
					},
				}, nil)
				h.schedulePublish(siteID)
			case "presence":
				cb(centrifuge.SubscribeReply{
					Options: centrifuge.SubscribeOptions{
						EmitPresence:  true,
						EmitJoinLeave: true,
					},
				}, nil)
				h.schedulePublish(siteID)
			default:
				cb(centrifuge.SubscribeReply{}, centrifuge.ErrorPermissionDenied)
			}
		})

		client.OnUnsubscribe(func(e centrifuge.UnsubscribeEvent) {
			if siteID, kind, ok := ChannelKind(e.Channel); ok && kind == "presence" {
				h.schedulePublish(siteID)
			}
		})
		client.OnDisconnect(func(_ centrifuge.DisconnectEvent) {
			if info, ok := parseConnInfo(client); ok {
				h.schedulePublish(info.SiteID)
			}
		})
	})
}

func (h *Hub) schedulePublish(siteID string) {
	h.publishMu.Lock()
	defer h.publishMu.Unlock()
	if timer, ok := h.publishAfter[siteID]; ok {
		timer.Stop()
	}
	h.publishAfter[siteID] = time.AfterFunc(300*time.Millisecond, func() {
		h.publishStats(siteID)
	})
}

func (h *Hub) publishStats(siteID string) {
	result, err := h.node.Presence(PresenceChannel(siteID))
	if err != nil {
		log.Printf("realtime presence for %s: %v", siteID, err)
		return
	}
	payload, err := statsPayload(result)
	if err != nil {
		log.Printf("realtime stats encode for %s: %v", siteID, err)
		return
	}
	_, err = h.node.Publish(AnalyticsChannel(siteID), payload, centrifuge.WithHistory(1, time.Minute))
	if err != nil {
		log.Printf("realtime publish for %s: %v", siteID, err)
	}
}

// WebSocketHandler returns the Centrifuge WebSocket HTTP handler.
func (h *Hub) WebSocketHandler() http.Handler {
	return h.ws
}

// Shutdown stops the Centrifuge node.
func (h *Hub) Shutdown(ctx context.Context) error {
	if h == nil || h.node == nil {
		return nil
	}
	return h.node.Shutdown(ctx)
}

// StatsForSite returns current presence stats without publishing.
func (h *Hub) StatsForSite(siteID string) (Stats, error) {
	result, err := h.node.Presence(PresenceChannel(siteID))
	if err != nil {
		return Stats{}, err
	}
	return buildStats(result), nil
}

// ConfigJSON returns connection settings for browser clients.
func ConfigJSON(siteID, scheme, host string, admin bool) ([]byte, error) {
	cfg := map[string]any{
		"endpoint": EndpointURL(scheme, host),
		"presence": PresenceChannel(siteID),
	}
	if admin {
		cfg["analytics"] = AnalyticsChannel(siteID)
	}
	return json.Marshal(cfg)
}
