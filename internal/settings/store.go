package settings

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/rob121/cannon/extension"
	"github.com/rob121/cannon/internal/models"
	"github.com/rob121/cannon/internal/sites"
	"gorm.io/gorm"
)

const ScopeGlobal = "global"

// Definition describes a global configuration section schema.
type Definition struct {
	ID       string
	Title    string
	Schema   json.RawMessage
	UISchema json.RawMessage
}

// Store loads and saves configuration values for a scope/section pair.
type Store interface {
	Load(ctx context.Context, scope, section string) (map[string]any, error)
	Save(ctx context.Context, scope, section string, data map[string]any) error
}

type dbStore struct{}

func (dbStore) Load(ctx context.Context, scope, section string) (map[string]any, error) {
	db, err := sites.DB(ctx)
	if err != nil {
		return nil, err
	}
	var row models.Setting
	err = db.Where("scope = ? AND section = ?", scope, section).First(&row).Error
	if err == gorm.ErrRecordNotFound {
		return map[string]any{}, nil
	}
	if err != nil {
		return nil, err
	}
	if row.Data == "" {
		return map[string]any{}, nil
	}
	var data map[string]any
	if err := json.Unmarshal([]byte(row.Data), &data); err != nil {
		return nil, fmt.Errorf("decode settings: %w", err)
	}
	return data, nil
}

func (dbStore) Save(ctx context.Context, scope, section string, data map[string]any) error {
	db, err := sites.DB(ctx)
	if err != nil {
		return err
	}
	raw, err := json.Marshal(data)
	if err != nil {
		return err
	}
	row := models.Setting{
		Scope:   scope,
		Section: section,
		Data:    string(raw),
	}
	return db.Save(&row).Error
}

// Document builds a configuration document for a scope from definitions and stored data.
func Document(ctx context.Context, store Store, scope string, defs []Definition) (extension.ConfigurationDocument, error) {
	doc := extension.ConfigurationDocument{Sections: make([]extension.ConfigurationSection, 0, len(defs))}
	for _, def := range defs {
		section := extension.ConfigurationSection{
			ID:       def.ID,
			Title:    def.Title,
			Schema:   def.Schema,
			UISchema: def.UISchema,
		}
		data, err := store.Load(ctx, scope, def.ID)
		if err != nil {
			return extension.ConfigurationDocument{}, err
		}
		if len(data) > 0 {
			raw, err := json.Marshal(data)
			if err != nil {
				return extension.ConfigurationDocument{}, err
			}
			section.Data = raw
		}
		doc.Sections = append(doc.Sections, section)
	}
	return doc, nil
}

// Save stores one section for a scope.
func Save(ctx context.Context, store Store, scope, section string, data map[string]any) error {
	if section == "" {
		return fmt.Errorf("configuration section is required")
	}
	return store.Save(ctx, scope, section, data)
}

// FindSection returns one section from a configuration document.
func FindSection(doc extension.ConfigurationDocument, sectionID string) (extension.ConfigurationSection, bool) {
	for _, section := range doc.Sections {
		if section.ID == sectionID {
			return section, true
		}
	}
	if sectionID == "" && len(doc.Sections) > 0 {
		return doc.Sections[0], true
	}
	return extension.ConfigurationSection{}, false
}
