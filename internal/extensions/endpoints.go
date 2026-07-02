package extensions

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"

	"github.com/rob121/cannon/extension"
	"github.com/rob121/cannon/internal/models"
)

func (m *Manager) fetchEndpoints(socketPath, endpointBase string) ([]extension.EndpointDefinition, error) {
	path := capabilityPath(endpointBase, "")
	resp, err := m.do(socketPath, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("status %d: %s", resp.StatusCode, string(body))
	}
	var payload extension.EndpointListResponse
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, err
	}
	return payload.Endpoints, nil
}

// DefaultEndpointDefinitions is used when GET /endpoint cannot be loaded.
var DefaultEndpointDefinitions = []extension.EndpointDefinition{
	{ID: "default", Title: "Default"},
}

func (m *Manager) loadEndpoints(socketPath, endpointBase string) []extension.EndpointDefinition {
	endpoints, err := m.fetchEndpoints(socketPath, endpointBase)
	if err == nil && len(endpoints) > 0 {
		return endpoints
	}
	return append([]extension.EndpointDefinition(nil), DefaultEndpointDefinitions...)
}

// EnsureEndpointDefinitions reloads endpoint lists for admin UI.
func (m *Manager) EnsureEndpointDefinitions(ctx context.Context) {
	for _, rt := range m.EndpointRuntimes() {
		m.mu.RLock()
		socket := rt.Model.Socket
		endpointBase := rt.Capabilities.Endpoint
		m.mu.RUnlock()
		endpoints := m.loadEndpoints(socket, endpointBase)
		m.mu.Lock()
		rt.Endpoints = endpoints
		m.mu.Unlock()
	}
	_ = ctx
}

// RuntimeEndpoints returns endpoint definitions for admin UI, with a safe default fallback.
func RuntimeEndpoints(rt *Runtime) []extension.EndpointDefinition {
	if rt == nil || len(rt.Endpoints) == 0 {
		return append([]extension.EndpointDefinition(nil), DefaultEndpointDefinitions...)
	}
	return rt.Endpoints
}

// EndpointRuntimes returns active extensions that expose /endpoint.
func (m *Manager) EndpointRuntimes() []*Runtime {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]*Runtime, 0, len(m.runtimes))
	for _, rt := range m.runtimes {
		if rt.Capabilities.Endpoint != "" && rt.Model.Status == models.StatusActive {
			out = append(out, rt)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Model.Sort < out[j].Model.Sort })
	return out
}

// InvokeEndpoint POSTs to an extension endpoint handler and returns the full wire response.
func (m *Manager) InvokeEndpoint(ctx context.Context, extensionName, item string, r *http.Request, userCtx, endpointData map[string]any) (WireResponse, error) {
	rt, ok := m.runtime(extensionName)
	if !ok || rt.Capabilities.Endpoint == "" {
		return WireResponse{}, fmt.Errorf("extension %s has no endpoint capability", extensionName)
	}
	item = strings.TrimSpace(item)
	if item == "" {
		item = "default"
	}
	wire, err := m.buildWireRequest(r, userCtx)
	if err != nil {
		return WireResponse{}, err
	}
	wire.EndpointItem = item
	if len(endpointData) > 0 {
		wire.EndpointData = endpointData
	}
	raw, _ := json.Marshal(wire)
	path := capabilityPath(rt.Capabilities.Endpoint, item)
	resp, err := m.do(rt.Model.Socket, http.MethodPost, path, bytes.NewReader(raw))
	if err != nil {
		return WireResponse{}, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	var out WireResponse
	if err := json.Unmarshal(body, &out); err != nil {
		return WireResponse{StatusCode: http.StatusOK, Body: string(body)}, nil
	}
	return out, nil
}
