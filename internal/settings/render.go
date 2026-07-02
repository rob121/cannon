package settings

import (
	"encoding/json"
	"fmt"
	"html"
	"net/http"
	"net/url"
	"regexp"
	"strings"

	gojsonforms "github.com/TobiEiss/go-jsonforms"
	"github.com/rob121/cannon/extension"
	"github.com/rob121/cannon/internal/csrf"
)

var booleanFieldRE = regexp.MustCompile(`(?s)<div class="form-group column col-12">\s*<label class="form-label admin-form-label" for="([^"]+)">([^<]*)</label>\s*<input class="form-input admin-form-control" id="([^"]+)" name="([^"]+)"\s+type="boolean"([^>]*)/>\s*(<small[^>]*>[\s\S]*?</small>)?\s*</div>`)

var selectWithoutNameRE = regexp.MustCompile(`<select class="form-select admin-form-control" id="([^"]+)" list="`)

var selectBlockRE = regexp.MustCompile(`(?s)(<select class="form-select admin-form-control" id="(#[^"]+)" name="[^"]+" list="[^"]*">)(.*?)(</select>)`)

var selectOptionRE = regexp.MustCompile(`<option value="([^"]*)">([^<]*)</option>`)

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
	data := parseSectionData(section.Data)
	html = normalizeSelectFields(html, data)
	html = normalizeBooleanFields(html, data)
	return injectDefaultPlaceholders(html, section.Schema, data)
}

var stringInputTagRE = regexp.MustCompile(`(?s)<input class="form-input admin-form-control" id="(#[^"]+)" name="([^"]+)"([^>]*)/>`)

func injectDefaultPlaceholders(formHTML string, schema json.RawMessage, data map[string]any) string {
	defaults := stringDefaultPlaceholders(schema)
	if len(defaults) == 0 {
		return formHTML
	}
	return stringInputTagRE.ReplaceAllStringFunc(formHTML, func(match string) string {
		parts := stringInputTagRE.FindStringSubmatch(match)
		if len(parts) < 4 || parts[1] != parts[2] {
			return match
		}
		attrs := parts[3]
		if strings.Contains(attrs, `value="`) || strings.Contains(attrs, `placeholder="`) {
			return match
		}
		key := propertyKey(parts[1])
		placeholder, ok := defaults[key]
		if !ok || formFieldValue(data, key) != "" {
			return match
		}
		return fmt.Sprintf(`<input class="form-input admin-form-control" id="%s" name="%s"%s placeholder="%s"/>`, parts[1], parts[2], attrs, html.EscapeString(placeholder))
	})
}

func stringDefaultPlaceholders(schema json.RawMessage) map[string]string {
	var doc struct {
		Properties map[string]struct {
			Type    string `json:"type"`
			Default any    `json:"default"`
		} `json:"properties"`
	}
	if err := json.Unmarshal(schema, &doc); err != nil {
		return nil
	}
	out := make(map[string]string, len(doc.Properties))
	for key, prop := range doc.Properties {
		if prop.Type != "string" || prop.Default == nil {
			continue
		}
		value := strings.TrimSpace(fmt.Sprint(prop.Default))
		if value != "" {
			out[key] = value
		}
	}
	return out
}

func normalizeSelectFields(html string, data map[string]any) string {
	html = selectWithoutNameRE.ReplaceAllString(html, `<select class="form-select admin-form-control" id="$1" name="$1" list="`)
	return selectBlockRE.ReplaceAllStringFunc(html, func(match string) string {
		parts := selectBlockRE.FindStringSubmatch(match)
		if len(parts) < 5 {
			return match
		}
		open, scope, inner, close := parts[1], parts[2], parts[3], parts[4]
		want := formFieldValue(data, propertyKey(scope))
		if want == "" {
			return match
		}
		inner = selectOptionRE.ReplaceAllStringFunc(inner, func(opt string) string {
			sub := selectOptionRE.FindStringSubmatch(opt)
			if len(sub) < 3 {
				return opt
			}
			value := sub[1]
			selected := ""
			if value == want {
				selected = " selected"
			}
			return fmt.Sprintf(`<option value="%s"%s>%s</option>`, value, selected, sub[2])
		})
		return open + inner + close
	})
}

func formFieldValue(data map[string]any, key string) string {
	if data == nil {
		return ""
	}
	v, ok := data[key]
	if !ok || v == nil {
		return ""
	}
	switch n := v.(type) {
	case float64:
		if n == float64(int64(n)) {
			return fmt.Sprintf("%d", int64(n))
		}
		return fmt.Sprint(n)
	case int:
		return fmt.Sprintf("%d", n)
	case int64:
		return fmt.Sprintf("%d", n)
	case bool:
		if n {
			return "true"
		}
		return "false"
	default:
		return strings.TrimSpace(fmt.Sprint(v))
	}
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
	if checked {
		checkedAttr = " checked"
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
    <span class="admin-form-toggle-text">
      <span class="admin-form-toggle-label-on">Enabled</span>
      <span class="admin-form-toggle-label-off">Disabled</span>
    </span>
  </label>%s
</div>`, scope, title, scope, scope, checkedAttr, smallBlock)
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
