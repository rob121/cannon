package themes

import (
	"encoding/json"
	"fmt"

	"github.com/rob121/cannon/extension"
)

// PatchGeneralSchema injects theme dropdown options into the general settings schema.
func PatchGeneralSchema(section extension.ConfigurationSection, templateDir string) (extension.ConfigurationSection, error) {
	frontend, err := FrontendOptions(templateDir)
	if err != nil {
		return section, err
	}
	admin, err := AdminOptions(templateDir)
	if err != nil {
		return section, err
	}

	var schema map[string]any
	if err := json.Unmarshal(section.Schema, &schema); err != nil {
		return section, err
	}
	props, _ := schema["properties"].(map[string]any)
	if props == nil {
		return section, fmt.Errorf("general schema missing properties")
	}
	patchEnum(props, "frontend_theme", frontend)
	patchEnum(props, "admin_theme", admin)
	raw, err := json.Marshal(schema)
	if err != nil {
		return section, err
	}
	section.Schema = raw
	return section, nil
}

func patchEnum(props map[string]any, key string, values []string) {
	field, ok := props[key].(map[string]any)
	if !ok {
		return
	}
	enum := make([]any, 0, len(values))
	for _, value := range values {
		enum = append(enum, value)
	}
	field["enum"] = enum
	props[key] = field
}
