package extensions

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

// Meta is extension metadata from GET /meta.
type Meta struct {
	Name          string `json:"name"`
	Title         string `json:"title"`
	Description   string `json:"description"`
	Version       string `json:"version"`
	UpdateURLBase string `json:"update_url_base"`
}

// MetaSummary is formatted meta data for admin display.
type MetaSummary struct {
	Available     bool
	Name          string
	Title         string
	Description   string
	Version       string
	UpdateURLBase string
	UpdateURL     string
}

// MetaSummary returns meta details for a running extension.
func (m *Manager) MetaSummary(name string) MetaSummary {
	rt, ok := m.runtime(name)
	if !ok || !m.IsRunning(name) || !rt.MetaAvailable {
		return MetaSummary{Available: false}
	}
	meta := rt.Meta
	return MetaSummary{
		Available:     true,
		Name:          meta.Name,
		Title:         meta.Title,
		Description:   meta.Description,
		Version:       meta.Version,
		UpdateURLBase: meta.UpdateURLBase,
		UpdateURL:     meta.UpdateURL(),
	}
}

func (meta Meta) UpdateURL() string {
	base := strings.TrimRight(strings.TrimSpace(meta.UpdateURLBase), "/")
	version := strings.TrimSpace(meta.Version)
	if base == "" || version == "" {
		return ""
	}
	return base + "/" + version
}

func (m *Manager) fetchMeta(socketPath string) (Meta, error) {
	resp, err := m.do(socketPath, http.MethodGet, "/meta", nil)
	if err != nil {
		return Meta{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return Meta{}, fmt.Errorf("meta: status %d", resp.StatusCode)
	}
	var payload Meta
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return Meta{}, err
	}
	return payload, nil
}
