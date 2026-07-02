package extensions

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"sort"
	"strings"

	"github.com/rob121/cannon/extension"
	"github.com/rob121/cannon/internal/hooks"
)

type HookWireRequest = extension.HookWireRequest
type HookWireResponse = extension.HookWireResponse

type hookListResponse struct {
	Hooks []string `json:"hooks"`
}

func (m *Manager) fetchHooks(socketPath, hookPath string) ([]string, error) {
	hookPath = normalizeHookPath(hookPath)
	resp, err := m.do(socketPath, http.MethodGet, hookPath, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var payload hookListResponse
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, err
	}
	return payload.Hooks, nil
}

func normalizeHookPath(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return "/hooks"
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	return strings.TrimRight(path, "/")
}

// FireHook runs in-process listeners and extension hook handlers for an event.
func (m *Manager) FireHook(ctx context.Context, r *http.Request, userCtx map[string]any, event string, args map[string]any) (map[string]any, error) {
	if args == nil {
		args = map[string]any{}
	}
	local, stop, err := hooks.FireLocal(ctx, r, event, args)
	if err != nil {
		return local, err
	}
	args = local
	if stop {
		return args, hookAbortErr(args)
	}

	m.mu.RLock()
	runtimes := make([]*Runtime, 0, len(m.runtimes))
	for _, rt := range m.runtimes {
		if rt.Capabilities.Hooks != "" && rt.subscribesHook(event) {
			runtimes = append(runtimes, rt)
		}
	}
	m.mu.RUnlock()
	sort.Slice(runtimes, func(i, j int) bool { return runtimes[i].Model.Sort < runtimes[j].Model.Sort })

	for _, rt := range runtimes {
		out, err := m.invokeHook(ctx, rt, event, r, userCtx, args)
		if err != nil {
			return args, err
		}
		if len(out.Arguments) > 0 {
			args = mergeHookArgs(args, out.Arguments)
		}
		if out.Stop {
			return args, hookAbortErr(args)
		}
	}
	return args, nil
}

func hookAbortErr(args map[string]any) error {
	if msg, ok := args["error"].(string); ok && msg != "" {
		return hooks.WrapAbort(msg)
	}
	if allowed, ok := args["allowed"].(bool); ok && !allowed {
		return hooks.ErrAborted
	}
	return nil
}

func (rt *Runtime) subscribesHook(event string) bool {
	for _, name := range rt.Hooks {
		if name == event {
			return true
		}
	}
	return false
}

func (m *Manager) invokeHook(ctx context.Context, rt *Runtime, event string, r *http.Request, userCtx map[string]any, args map[string]any) (HookWireResponse, error) {
	wire, err := m.buildWireRequest(r, userCtx)
	if err != nil {
		return HookWireResponse{}, err
	}
	payload := HookWireRequest{
		WireRequest: *wire,
		Event:       event,
		Arguments:   args,
	}
	raw, _ := json.Marshal(payload)
	path := normalizeHookPath(rt.Capabilities.Hooks)
	resp, err := m.do(rt.Model.Socket, http.MethodPost, path, bytes.NewReader(raw))
	if err != nil {
		return HookWireResponse{}, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	var out HookWireResponse
	if err := json.Unmarshal(body, &out); err != nil {
		return HookWireResponse{WireResponse: extension.WireResponse{StatusCode: http.StatusOK, Body: string(body)}}, nil
	}
	if out.StatusCode >= 400 && out.Body != "" {
		return out, hooks.WrapAbort(out.Body)
	}
	return out, nil
}

func mergeHookArgs(base, patch map[string]any) map[string]any {
	out := map[string]any{}
	for k, v := range base {
		out[k] = v
	}
	for k, v := range patch {
		out[k] = v
	}
	return out
}
