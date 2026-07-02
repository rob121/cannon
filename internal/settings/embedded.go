package settings

import (
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"path/filepath"
	"sort"
	"strings"
)

//go:embed definitions/*.json
var definitionFS embed.FS

type embeddedDefinition struct {
	Title    string          `json:"title"`
	Schema   json.RawMessage `json:"schema"`
	UISchema json.RawMessage `json:"ui_schema"`
}

// GlobalDefinitions returns global configuration sections embedded in the Cannon binary.
func GlobalDefinitions() ([]Definition, error) {
	entries, err := fs.ReadDir(definitionFS, "definitions")
	if err != nil {
		return nil, err
	}

	defs := make([]Definition, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		def, err := loadEmbeddedDefinition(entry.Name())
		if err != nil {
			return nil, err
		}
		defs = append(defs, def)
	}
	sort.Slice(defs, func(i, j int) bool { return defs[i].ID < defs[j].ID })
	return defs, nil
}

// GlobalDefinition returns one embedded global section by id.
func GlobalDefinition(sectionID string) (Definition, bool, error) {
	defs, err := GlobalDefinitions()
	if err != nil {
		return Definition{}, false, err
	}
	for _, def := range defs {
		if def.ID == sectionID {
			return def, true, nil
		}
	}
	return Definition{}, false, nil
}

func loadEmbeddedDefinition(name string) (Definition, error) {
	raw, err := definitionFS.ReadFile(filepath.Join("definitions", name))
	if err != nil {
		return Definition{}, err
	}
	var payload embeddedDefinition
	if err := json.Unmarshal(raw, &payload); err != nil {
		return Definition{}, fmt.Errorf("parse embedded definition %s: %w", name, err)
	}
	if payload.Title == "" {
		return Definition{}, fmt.Errorf("embedded definition %s: title is required", name)
	}
	if len(payload.Schema) == 0 || len(payload.UISchema) == 0 {
		return Definition{}, fmt.Errorf("embedded definition %s: schema and ui_schema are required", name)
	}
	id := strings.TrimSuffix(name, filepath.Ext(name))
	return Definition{
		ID:       id,
		Title:    payload.Title,
		Schema:   payload.Schema,
		UISchema: payload.UISchema,
	}, nil
}
