package extensions

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/rob121/cannon/extension"
)

// FetchConfiguration loads GET /configuration from a running extension.
func (m *Manager) FetchConfiguration(socketPath string) (extension.ConfigurationDocument, error) {
	resp, err := m.do(socketPath, http.MethodGet, "/configuration", nil)
	if err != nil {
		return extension.ConfigurationDocument{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return extension.ConfigurationDocument{}, fmt.Errorf("status %d: %s", resp.StatusCode, string(body))
	}
	var doc extension.ConfigurationDocument
	if err := json.NewDecoder(resp.Body).Decode(&doc); err != nil {
		return extension.ConfigurationDocument{}, err
	}
	return doc, nil
}

// SaveConfiguration posts updated section data to an extension.
func (m *Manager) SaveConfiguration(socketPath string, req extension.ConfigurationSaveRequest) error {
	raw, err := json.Marshal(req)
	if err != nil {
		return err
	}
	resp, err := m.do(socketPath, http.MethodPost, "/configuration", bytes.NewReader(raw))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 300 {
		return fmt.Errorf("status %d: %s", resp.StatusCode, string(body))
	}
	var out map[string]string
	if err := json.Unmarshal(body, &out); err == nil {
		if msg := out["error"]; msg != "" {
			return fmt.Errorf("%s", msg)
		}
	}
	return nil
}

// ConfigurationRuntimes returns active extensions that expose /configuration.
func (m *Manager) ConfigurationRuntimes() []*Runtime {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]*Runtime, 0)
	for _, rt := range m.runtimes {
		if rt.Capabilities.Configuration != "" {
			out = append(out, rt)
		}
	}
	return out
}
