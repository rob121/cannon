package themes

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/rob121/cannon/extension"
)

// PatchGeneralSchema injects theme and captcha dropdown options into the general settings schema.
func PatchGeneralSchema(section extension.ConfigurationSection, templateDir string, captchaExtensions []string) (extension.ConfigurationSection, error) {
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
	patchCaptchaExtensionEnum(props, captchaExtensions, currentSettingString(section.Data, "captcha_active_extension"))
	raw, err := json.Marshal(schema)
	if err != nil {
		return section, err
	}
	section.Schema = raw
	return section, nil
}

func currentSettingString(data json.RawMessage, key string) string {
	if len(data) == 0 {
		return ""
	}
	var values map[string]any
	if err := json.Unmarshal(data, &values); err != nil {
		return ""
	}
	v, ok := values[key]
	if !ok || v == nil {
		return ""
	}
	return strings.TrimSpace(fmt.Sprint(v))
}

func patchCaptchaExtensionEnum(props map[string]any, names []string, current string) {
	field, ok := props["captcha_active_extension"].(map[string]any)
	if !ok {
		return
	}
	enum := []any{""}
	seen := map[string]bool{"": true}
	for _, name := range names {
		name = strings.TrimSpace(name)
		if name == "" || seen[name] {
			continue
		}
		enum = append(enum, name)
		seen[name] = true
	}
	current = strings.TrimSpace(current)
	if current != "" && !seen[current] {
		enum = append(enum, current)
	}
	field["enum"] = enum
	props["captcha_active_extension"] = field
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
