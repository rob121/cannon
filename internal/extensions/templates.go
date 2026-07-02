package extensions

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/rob121/cannon/extension"
)

// TemplateItem describes an extension template for admin display.
type TemplateItem struct {
	extension.TemplateDefinition
	Overridden bool
}

// TemplateSummary is formatted extension template data for admin display.
type TemplateSummary struct {
	Available bool
	Items     []TemplateItem
}

func (m *Manager) fetchTemplates(socketPath, templateBase string) ([]extension.TemplateDefinition, error) {
	path := capabilityPath(templateBase, "")
	resp, err := m.do(socketPath, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("status %d: %s", resp.StatusCode, string(body))
	}
	var payload extension.TemplateListResponse
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, err
	}
	return payload.Templates, nil
}

// TemplateSource fetches an embedded extension template by local path.
func (m *Manager) TemplateSource(extensionName, path string) (extension.TemplateSourceResponse, error) {
	rt, ok := m.runtime(extensionName)
	if !ok || rt.Capabilities.Templates == "" {
		return extension.TemplateSourceResponse{}, fmt.Errorf("extension %s has no templates capability", extensionName)
	}
	path = strings.Trim(path, "/")
	if path == "" {
		return extension.TemplateSourceResponse{}, fmt.Errorf("template path is required")
	}
	resp, err := m.do(rt.Model.Socket, http.MethodGet, capabilityPath(rt.Capabilities.Templates, path), nil)
	if err != nil {
		return extension.TemplateSourceResponse{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return extension.TemplateSourceResponse{}, fmt.Errorf("status %d: %s", resp.StatusCode, string(body))
	}
	var payload extension.TemplateSourceResponse
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return extension.TemplateSourceResponse{}, err
	}
	return payload, nil
}

// TemplateSummary returns overridable templates advertised by a running extension.
func (m *Manager) TemplateSummary(extensionName, templateDir string) TemplateSummary {
	rt, ok := m.runtime(extensionName)
	if !ok || !m.IsRunning(extensionName) || rt.Capabilities.Templates == "" {
		return TemplateSummary{Available: false}
	}
	items := make([]TemplateItem, 0, len(rt.Templates))
	for _, def := range rt.Templates {
		item := TemplateItem{TemplateDefinition: def}
		if templateDir != "" && def.OverridePath != "" {
			if _, err := os.Stat(filepath.Join(templateDir, filepath.FromSlash(def.OverridePath))); err == nil {
				item.Overridden = true
			}
		}
		items = append(items, item)
	}
	sort.Slice(items, func(i, j int) bool { return items[i].Path < items[j].Path })
	return TemplateSummary{Available: true, Items: items}
}
