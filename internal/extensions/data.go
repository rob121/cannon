package extensions

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/rob121/cannon/internal/models"
)

const ExtensionDataPrefix = "/ext/"

// ParseExtensionDataPath splits /ext/{route_hash}/{data_path} into hash and data path.
func ParseExtensionDataPath(path string) (hash, dataPath string, ok bool) {
	path = strings.TrimSpace(path)
	if !strings.HasPrefix(path, ExtensionDataPrefix) {
		return "", "", false
	}
	rest := strings.TrimPrefix(path, ExtensionDataPrefix)
	rest = strings.Trim(rest, "/")
	if rest == "" {
		return "", "", false
	}
	parts := strings.SplitN(rest, "/", 2)
	hash = strings.TrimSpace(parts[0])
	if hash == "" {
		return "", "", false
	}
	if len(parts) == 2 {
		dataPath = strings.Trim(parts[1], "/")
	}
	if dataPath == "" {
		return "", "", false
	}
	return hash, dataPath, true
}

// RuntimeByRouteHash finds a running extension by its public route hash.
func (m *Manager) RuntimeByRouteHash(hash string) (*Runtime, bool) {
	hash = strings.ToLower(strings.TrimSpace(hash))
	if hash == "" {
		return nil, false
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, rt := range m.runtimes {
		if rt.Model.Status != models.StatusActive {
			continue
		}
		if strings.EqualFold(RouteHashFromSocket(rt.Model.Socket), hash) {
			return rt, true
		}
		if strings.EqualFold(routeHashHex(rt.Model.Name, m.site.ID), hash) {
			return rt, true
		}
	}
	return nil, false
}

// InvokeData POSTs to an extension data handler and returns the full wire response.
func (m *Manager) InvokeData(ctx context.Context, extensionName, dataPath string, r *http.Request, userCtx map[string]any) (WireResponse, error) {
	rt, ok := m.runtime(extensionName)
	if !ok || rt.Capabilities.Data == "" {
		return WireResponse{}, fmt.Errorf("extension %s has no data capability", extensionName)
	}
	dataPath = strings.Trim(dataPath, "/")
	if dataPath == "" {
		return WireResponse{}, fmt.Errorf("data path is required")
	}
	wire, err := m.buildWireRequest(r, userCtx)
	if err != nil {
		return WireResponse{}, err
	}
	wire.DataPath = dataPath
	raw, _ := json.Marshal(wire)
	path := capabilityPath(rt.Capabilities.Data, dataPath)
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

// ServeExtensionData handles /ext/{route_hash}/{data_path} if the extension supports /data.
func (m *Manager) ServeExtensionData(ctx context.Context, w http.ResponseWriter, r *http.Request, userCtx map[string]any) (bool, error) {
	hash, dataPath, ok := ParseExtensionDataPath(r.URL.Path)
	if !ok {
		return false, nil
	}
	rt, ok := m.RuntimeByRouteHash(hash)
	if !ok || rt.Capabilities.Data == "" {
		http.NotFound(w, r)
		return true, nil
	}
	out, err := m.InvokeData(ctx, rt.Model.Name, dataPath, r, userCtx)
	if err != nil {
		return true, err
	}
	WriteHTTPResponse(w, out)
	return true, nil
}
