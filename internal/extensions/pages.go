package extensions

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strings"

	"github.com/rob121/cannon/extension"
	"github.com/rob121/cannon/internal/models"
)

func (m *Manager) fetchPages(socketPath, pageBase string) ([]extension.PageDefinition, error) {
	path := capabilityPath(pageBase, "")
	resp, err := m.do(socketPath, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("status %d: %s", resp.StatusCode, string(body))
	}
	var payload extension.PageListResponse
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, err
	}
	return payload.Pages, nil
}

func capabilityPath(base, item string) string {
	path := "/" + strings.TrimPrefix(strings.TrimSpace(base), "/")
	path = strings.TrimSuffix(path, "/")
	if item == "" {
		return path
	}
	return path + "/" + url.PathEscape(item)
}

// DefaultPageDefinitions is used when GET /page cannot be loaded.
var DefaultPageDefinitions = []extension.PageDefinition{
	{ID: "default", Title: "Default"},
}

func (m *Manager) loadPages(socketPath, pageBase string) []extension.PageDefinition {
	pages, err := m.fetchPages(socketPath, pageBase)
	if err == nil && len(pages) > 0 {
		return pages
	}
	return append([]extension.PageDefinition(nil), DefaultPageDefinitions...)
}

// EnsurePageDefinitions reloads page lists for admin UI.
func (m *Manager) EnsurePageDefinitions(ctx context.Context) {
	for _, rt := range m.PageRuntimes() {
		m.mu.RLock()
		socket := rt.Model.Socket
		pageBase := rt.Capabilities.Page
		m.mu.RUnlock()
		pages := m.loadPages(socket, pageBase)
		m.mu.Lock()
		rt.Pages = pages
		m.mu.Unlock()
	}
	_ = ctx
}

// RuntimePages returns page definitions for admin UI, with a safe default fallback.
func RuntimePages(rt *Runtime) []extension.PageDefinition {
	if rt == nil || len(rt.Pages) == 0 {
		return append([]extension.PageDefinition(nil), DefaultPageDefinitions...)
	}
	return rt.Pages
}

// PageRuntimes returns active extensions that expose /page.
func (m *Manager) PageRuntimes() []*Runtime {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]*Runtime, 0, len(m.runtimes))
	for _, rt := range m.runtimes {
		if rt.Capabilities.Page != "" && rt.Model.Status == models.StatusActive {
			out = append(out, rt)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Model.Sort < out[j].Model.Sort })
	return out
}
