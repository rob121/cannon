package settings

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strings"

	gojsonforms "github.com/TobiEiss/go-jsonforms"
	"github.com/rob121/cannon/extension"
	"github.com/rob121/cannon/internal/csrf"
)

var booleanFieldRE = regexp.MustCompile(`(?s)<div class="form-group column col-12">\s*<label class="form-label admin-form-label" for="([^"]+)">([^<]*)</label>\s*<input class="form-input admin-form-control" id="([^"]+)" name="([^"]+)"\s+type="boolean"([^>]*)/>\s*(<small[^>]*>[\s\S]*?</small>)?\s*</div>`)

// RenderForm builds HTML for a JSON Forms section using admin styling hooks.
func RenderForm(section extension.ConfigurationSection, postURL, csrfToken string) (string, error) {
	builder := gojsonforms.NewBuilder().
		WithSchemaBytes(section.Schema).
		WithUISchemaBytes(section.UISchema)
	if len(section.Data) > 0 {
		builder = builder.WithDataBytes(section.Data)
	}
	postLink := strings.TrimPrefix(strings.TrimSpace(postURL), "/")
	html, err := builder.WithPostLink(postLink).Build(false)
	if err != nil {
		return "", err
	}
	html = injectCSRFField(html, csrfToken)
	return normalizeFormHTML(html, section), nil
}

func injectCSRFField(html, token string) string {
	field := string(csrf.HiddenField(token))
	if field == "" {
		return html
	}
	idx := strings.Index(html, "<form")
	if idx < 0 {
		return html
	}
	end := strings.Index(html[idx:], ">")
	if end < 0 {
		return html
	}
	insert := idx + end + 1
	return html[:insert] + field + html[insert:]
}

// FormDataFromRequest parses posted configuration values and applies boolean defaults.
func FormDataFromRequest(r *http.Request, schema json.RawMessage) map[string]any {
	data := map[string]any{}
	if result := gojsonforms.Verify(r.Form); result != nil {
		if parsed, ok := result.(map[string]any); ok {
			data = parsed
		}
	}
	return ApplyBooleanFormDefaults(schema, data, r.Form)
}

func normalizeFormHTML(html string, section extension.ConfigurationSection) string {
	replacements := []struct{ old, new string }{
		{`class="form-input"`, `class="form-input admin-form-control"`},
		{`class="form-select"`, `class="form-select admin-form-control"`},
		{`class="form-label"`, `class="form-label admin-form-label"`},
		{`class="btn" type="submit"`, `class="btn btn-admin-primary" type="submit"`},
		{`class="card"`, `class="card admin-jsonforms-card"`},
		{`class="card-header"`, `class="card-header admin-jsonforms-card-header"`},
		{`class="card-body"`, `class="card-body admin-jsonforms-card-body"`},
	}
	for _, r := range replacements {
		html = strings.ReplaceAll(html, r.old, r.new)
	}
	return normalizeBooleanFields(html, parseSectionData(section.Data))
}

func normalizeBooleanFields(html string, data map[string]any) string {
	return booleanFieldRE.ReplaceAllStringFunc(html, func(match string) string {
		parts := booleanFieldRE.FindStringSubmatch(match)
		if len(parts) < 5 {
			return match
		}
		scope := parts[1]
		title := parts[2]
		smallHTML := ""
		if len(parts) > 6 {
			smallHTML = parts[6]
		}
		return renderBooleanToggle(scope, title, smallHTML, data)
	})
}

func renderBooleanToggle(scope, title, smallHTML string, data map[string]any) string {
	key := propertyKey(scope)
	checked := Bool(data, key)
	checkedAttr := ""
	text := "Disabled"
	if checked {
		checkedAttr = " checked"
		text = "Enabled"
	}
	smallBlock := ""
	if strings.TrimSpace(smallHTML) != "" {
		smallBlock = "\n  " + smallHTML
	}
	return fmt.Sprintf(`<div class="form-group column col-12">
  <label class="form-label admin-form-label" for="%s">%s</label>
  <label class="admin-form-toggle admin-jsonforms-toggle">
    <input type="checkbox" class="admin-form-toggle-input" id="%s" name="%s" value="true"%s>
    <span class="admin-form-toggle-track" aria-hidden="true"><span class="admin-form-toggle-thumb"></span></span>
    <span class="admin-form-toggle-text" data-on="Enabled" data-off="Disabled">%s</span>
  </label>%s
</div>`, scope, title, scope, scope, checkedAttr, text, smallBlock)
}

func parseSectionData(raw json.RawMessage) map[string]any {
	if len(raw) == 0 {
		return map[string]any{}
	}
	var data map[string]any
	if err := json.Unmarshal(raw, &data); err != nil {
		return map[string]any{}
	}
	return data
}

func propertyKey(scope string) string {
	return strings.TrimPrefix(scope, "#/properties/")
}

// ApplyBooleanFormDefaults sets false for unchecked boolean fields in a JSON Forms POST.
func ApplyBooleanFormDefaults(schema json.RawMessage, data map[string]any, form url.Values) map[string]any {
	if data == nil {
		data = map[string]any{}
	}
	for _, key := range booleanPropertyKeys(schema) {
		formKey := "#/properties/" + key
		if form.Has(formKey) {
			data[key] = formValueBool(form.Get(formKey))
			continue
		}
		data[key] = false
	}
	return data
}

func booleanPropertyKeys(schema json.RawMessage) []string {
	var doc struct {
		Properties map[string]struct {
			Type string `json:"type"`
		} `json:"properties"`
	}
	if err := json.Unmarshal(schema, &doc); err != nil {
		return nil
	}
	keys := make([]string, 0, len(doc.Properties))
	for key, prop := range doc.Properties {
		if prop.Type == "boolean" {
			keys = append(keys, key)
		}
	}
	return keys
}

func formValueBool(raw string) bool {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "1", "true", "on", "yes":
		return true
	default:
		return false
	}
}
