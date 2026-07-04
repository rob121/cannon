package settings

import "encoding/json"

// categoryField describes a configuration property rendered as a category dropdown.
type categoryField struct {
	Key   string
	Scope string
}

func categoryFields(schema, uiSchema json.RawMessage) []categoryField {
	controls := listControls(uiSchema)
	if len(controls) == 0 {
		return nil
	}
	formats := propertyFormats(schema)
	var fields []categoryField
	seen := map[string]bool{}
	for _, ctrl := range controls {
		key := propertyKey(ctrl.Scope)
		if key == "" || seen[key] {
			continue
		}
		if ctrl.Format == "category" || formats[key] == "category" {
			fields = append(fields, categoryField{Key: key, Scope: ctrl.Scope})
			seen[key] = true
		}
	}
	return fields
}

type uiControl struct {
	Scope  string
	Format string
}

func listControls(uiSchema json.RawMessage) []uiControl {
	var node map[string]any
	if err := json.Unmarshal(uiSchema, &node); err != nil {
		return nil
	}
	var controls []uiControl
	collectControls(node, &controls)
	return controls
}

func collectControls(node map[string]any, controls *[]uiControl) {
	if node == nil {
		return
	}
	if nodeType, _ := node["type"].(string); nodeType == "Control" {
		if scope, _ := node["scope"].(string); scope != "" {
			ctrl := uiControl{Scope: scope}
			if options, ok := node["options"].(map[string]any); ok {
				if format, _ := options["format"].(string); format != "" {
					ctrl.Format = format
				}
			}
			*controls = append(*controls, ctrl)
		}
	}
	if elements, ok := node["elements"].([]any); ok {
		for _, el := range elements {
			if child, ok := el.(map[string]any); ok {
				collectControls(child, controls)
			}
		}
	}
}

func propertyAllowsNull(schema json.RawMessage, key string) bool {
	var doc struct {
		Properties map[string]struct {
			Type any `json:"type"`
		} `json:"properties"`
	}
	if err := json.Unmarshal(schema, &doc); err != nil {
		return false
	}
	prop, ok := doc.Properties[key]
	if !ok {
		return false
	}
	switch typed := prop.Type.(type) {
	case string:
		return typed == "null"
	case []any:
		for _, item := range typed {
			if fmt, ok := item.(string); ok && fmt == "null" {
				return true
			}
		}
	}
	return false
}

func propertyFormats(schema json.RawMessage) map[string]string {
	var doc struct {
		Properties map[string]struct {
			Format string `json:"format"`
		} `json:"properties"`
	}
	if err := json.Unmarshal(schema, &doc); err != nil {
		return nil
	}
	out := make(map[string]string, len(doc.Properties))
	for key, prop := range doc.Properties {
		if prop.Format != "" {
			out[key] = prop.Format
		}
	}
	return out
}

// normalizeSchemaForForms adapts JSON Schema for go-jsonforms, which does not accept type arrays.
func normalizeSchemaForForms(schema json.RawMessage) json.RawMessage {
	var doc map[string]any
	if err := json.Unmarshal(schema, &doc); err != nil {
		return schema
	}
	props, _ := doc["properties"].(map[string]any)
	for key, raw := range props {
		prop, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		prop["type"] = primaryJSONType(prop["type"])
		props[key] = prop
	}
	doc["properties"] = props
	out, err := json.Marshal(doc)
	if err != nil {
		return schema
	}
	return out
}

func primaryJSONType(raw any) any {
	switch typed := raw.(type) {
	case string:
		return typed
	case []any:
		for _, item := range typed {
			if s, ok := item.(string); ok && s != "null" {
				return s
			}
		}
		if len(typed) > 0 {
			return typed[0]
		}
	}
	return raw
}
