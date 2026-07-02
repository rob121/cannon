package extension

import (
	"encoding/json"
	"fmt"
)

const configurationPath = "/configuration"

// ConfigurationSection is one JSON Forms section returned from GET /configuration.
type ConfigurationSection struct {
	ID       string          `json:"id"`
	Title    string          `json:"title"`
	Schema   json.RawMessage `json:"schema"`
	UISchema json.RawMessage `json:"ui_schema"`
	Data     json.RawMessage `json:"data,omitempty"`
}

// ConfigurationDocument is the GET /configuration response body.
type ConfigurationDocument struct {
	Sections []ConfigurationSection `json:"sections"`
}

// ConfigurationSaveRequest is the POST /configuration request body.
type ConfigurationSaveRequest struct {
	Section string          `json:"section"`
	Data    json.RawMessage `json:"data"`
}

// ConfigurationDefinition describes a section without stored data.
type ConfigurationDefinition struct {
	ID       string
	Title    string
	Schema   json.RawMessage
	UISchema json.RawMessage
}

// ConfigurationStore loads and persists section data for an extension.
type ConfigurationStore interface {
	Load(section string) (map[string]any, error)
	Save(section string, data map[string]any) error
}

// ConfigurationProvider serves extension settings over /configuration.
type ConfigurationProvider interface {
	Configuration() (ConfigurationDocument, error)
	SaveConfiguration(req ConfigurationSaveRequest) error
}

// MapConfiguration builds a provider from static definitions and a store.
func MapConfiguration(defs []ConfigurationDefinition, store ConfigurationStore) ConfigurationProvider {
	return &mapConfiguration{defs: defs, store: store}
}

type mapConfiguration struct {
	defs  []ConfigurationDefinition
	store ConfigurationStore
}

func (m *mapConfiguration) Configuration() (ConfigurationDocument, error) {
	doc := ConfigurationDocument{Sections: make([]ConfigurationSection, 0, len(m.defs))}
	for _, def := range m.defs {
		section := ConfigurationSection{
			ID:       def.ID,
			Title:    def.Title,
			Schema:   def.Schema,
			UISchema: def.UISchema,
		}
		if m.store != nil {
			data, err := m.store.Load(def.ID)
			if err != nil {
				return ConfigurationDocument{}, err
			}
			if data != nil {
				raw, err := json.Marshal(data)
				if err != nil {
					return ConfigurationDocument{}, err
				}
				section.Data = raw
			}
		}
		doc.Sections = append(doc.Sections, section)
	}
	return doc, nil
}

func (m *mapConfiguration) SaveConfiguration(req ConfigurationSaveRequest) error {
	if req.Section == "" {
		return fmt.Errorf("configuration section is required")
	}
	if m.store == nil {
		return fmt.Errorf("configuration store is not configured")
	}
	var data map[string]any
	if len(req.Data) > 0 {
		if err := json.Unmarshal(req.Data, &data); err != nil {
			return fmt.Errorf("invalid configuration data: %w", err)
		}
	}
	return m.store.Save(req.Section, data)
}
