package extension

import (
	"encoding/json"
	"net/http"
	"strings"
)

// HookWireRequest is the JSON payload Cannon POSTs to extension hook handlers.
type HookWireRequest struct {
	WireRequest
	Event     string         `json:"event"`
	Arguments map[string]any `json:"arguments"`
}

// HookWireResponse is returned by extension hook handlers.
type HookWireResponse struct {
	WireResponse
	Arguments map[string]any `json:"arguments,omitempty"`
}

// HookHandler handles one hook event.
type HookHandler func(req HookWireRequest) HookWireResponse

// HookListResponse is returned from GET /hooks.
type HookListResponse struct {
	Hooks []string `json:"hooks"`
}

// HookOK returns a successful hook response with optional argument updates.
func HookOK(args map[string]any) HookWireResponse {
	return HookWireResponse{WireResponse: OK(), Arguments: args}
}

// HookStop stops further hook listeners with optional argument updates.
func HookStop(args map[string]any) HookWireResponse {
	return HookWireResponse{WireResponse: WireResponse{Stop: true}, Arguments: args}
}

// HookAbort blocks the operation (for example login) with a message in arguments.
func HookAbort(message string) HookWireResponse {
	args := map[string]any{"allowed": false}
	if message != "" {
		args["error"] = message
	}
	return HookWireResponse{
		WireResponse: WireResponse{Stop: true},
		Arguments:    args,
	}
}

// HookEvent returns the event name from a hook wire request.
func HookEvent(req HookWireRequest) string {
	return req.Event
}


// HookArguments returns hook arguments, never nil.
func HookArguments(req HookWireRequest) map[string]any {
	if req.Arguments == nil {
		return map[string]any{}
	}
	return req.Arguments
}

// HookString reads a string hook argument.
func HookString(args map[string]any, key string) string {
	v, ok := args[key].(string)
	if !ok {
		return ""
	}
	return strings.TrimSpace(v)
}

// HookHeadHTML returns accumulated head_html from hook arguments.
func HookHeadHTML(args map[string]any) string {
	return HookString(args, "head_html")
}

// HookBodyHTML returns accumulated body_html from hook arguments.
func HookBodyHTML(args map[string]any) string {
	return HookString(args, "body_html")
}

// HookRobotsAppend returns extra robots.txt lines from hook arguments.
func HookRobotsAppend(args map[string]any) string {
	return HookString(args, "robots_append")
}

// HookSitemapURLs returns extra sitemap entries from hook arguments.
func HookSitemapURLs(args map[string]any) []map[string]any {
	switch rows := args["sitemap_urls"].(type) {
	case []map[string]any:
		return append([]map[string]any(nil), rows...)
	case []any:
		out := make([]map[string]any, 0, len(rows))
		for _, row := range rows {
			if m, ok := row.(map[string]any); ok {
				out = append(out, m)
			}
		}
		return out
	default:
		return nil
	}
}

// DecodeHookWireRequest reads a hook wire request from an HTTP request body.
func DecodeHookWireRequest(r *http.Request, fallbackSiteID string) (HookWireRequest, error) {
	var req HookWireRequest
	if r.Body != nil && r.ContentLength != 0 {
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			return HookWireRequest{}, err
		}
	}
	if req.SiteID == "" {
		req.SiteID = fallbackSiteID
	}
	return req, nil
}
