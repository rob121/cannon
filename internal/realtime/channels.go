package realtime

import (
	"fmt"
	"strings"
)

const channelPrefix = "site:"

// PresenceChannel returns the public presence channel for a site.
func PresenceChannel(siteID string) string {
	return channelPrefix + siteID + ":presence"
}

// AnalyticsChannel returns the admin-only analytics feed channel for a site.
func AnalyticsChannel(siteID string) string {
	return channelPrefix + siteID + ":analytics"
}

// SiteIDFromChannel extracts the site id from a realtime channel name.
func SiteIDFromChannel(channel string) (string, bool) {
	if !strings.HasPrefix(channel, channelPrefix) {
		return "", false
	}
	rest := strings.TrimPrefix(channel, channelPrefix)
	parts := strings.SplitN(rest, ":", 2)
	if len(parts) != 2 || parts[0] == "" {
		return "", false
	}
	return parts[0], true
}

// ChannelKind classifies a realtime channel.
func ChannelKind(channel string) (siteID string, kind string, ok bool) {
	siteID, ok = SiteIDFromChannel(channel)
	if !ok {
		return "", "", false
	}
	switch {
	case strings.HasSuffix(channel, ":presence"):
		return siteID, "presence", true
	case strings.HasSuffix(channel, ":analytics"):
		return siteID, "analytics", true
	default:
		return "", "", false
	}
}

// WebSocketPath is the public WebSocket endpoint path.
const WebSocketPath = "/connection/websocket"

// EndpointURL builds a browser WebSocket URL for the current host.
func EndpointURL(scheme, host string) string {
	proto := "ws"
	if scheme == "https" || scheme == "wss" {
		proto = "wss"
	}
	return fmt.Sprintf("%s://%s%s", proto, host, WebSocketPath)
}
