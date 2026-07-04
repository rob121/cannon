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
	"github.com/rob121/cannon/internal/models"
)

// FormRenderContext supplies optional data for specialized configuration field types.
type FormRenderContext struct {
	Categories []models.Category
}

var booleanFieldRE = regexp.MustCompile(`(?s)<div class="form-group( column col-12)?">\s*<label class="form-label admin-form-label" for="([^"]+)">([^<]*)</label>\s*<input class="form-input admin-form-control" id="([^"]+)" name="([^"]+)"\s+type="boolean"([^>]*)/>\s*(<small[^>]*>[\s\S]*?</small>)?\s*</div>`)

var selectWithoutNameRE = regexp.MustCompile(`<select class="form-select admin-form-control" id="([^"]+)" list="`)

var selectBlockRE = regexp.MustCompile(`(?s)(<select class="form-select admin-form-control" id="(#[^"]+)" name="[^"]+" list="[^"]*">)(.*?)(</select>)`)

var selectOptionRE = regexp.MustCompile(`<option value="([^"]*)">([^<]*)</option>`)

// RenderForm builds HTML for a JSON Forms section using admin styling hooks.
func RenderForm(section extension.ConfigurationSection, postURL, csrfToken string, renderCtx *FormRenderContext) (string, error) {
	schema := normalizeSchemaForForms(section.Schema)
	builder := gojsonforms.NewBuilder().
		WithSchemaBytes(schema).
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
	return normalizeFormHTML(html, section, renderCtx), nil
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
func FormDataFromRequest(r *http.Request, schema, uiSchema json.RawMessage) map[string]any {
	data := map[string]any{}
	if result := gojsonforms.Verify(r.Form); result != nil {
		if parsed, ok := result.(map[string]any); ok {
			data = parsed
		}
	}
	data = ApplyBooleanFormDefaults(schema, data, r.Form)
	return ApplyCategoryFormDefaults(schema, uiSchema, data, r.Form)
}

func normalizeFormHTML(html string, section extension.ConfigurationSection, renderCtx *FormRenderContext) string {
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
	html = normalizeTextareaFields(html, section.UISchema, data)
	categories := []models.Category(nil)
	if renderCtx != nil {
		categories = renderCtx.Categories
	}
	html = normalizeCategoryFields(html, section.Schema, section.UISchema, data, categories)
	return injectDefaultPlaceholders(html, section.Schema, data)
}

var stringInputTagRE = regexp.MustCompile(`(?s)<input class="form-input admin-form-control" id="(#[^"]+)" name="([^"]+)"([^>]*)/>`)

func normalizeTextareaFields(formHTML string, uiSchema json.RawMessage, data map[string]any) string {
	for _, scope := range textareaScopes(uiSchema) {
		key := propertyKey(scope)
		value := html.EscapeString(formFieldValue(data, key))
		scopeRE := regexp.MustCompile(`(?s)<input class="form-input admin-form-control" id="` + regexp.QuoteMeta(scope) + `" name="` + regexp.QuoteMeta(scope) + `"([^>]*)/>`)
		formHTML = scopeRE.ReplaceAllStringFunc(formHTML, func(match string) string {
			aria := ""
			if m := regexp.MustCompile(`aria-describedby="([^"]*)"`).FindStringSubmatch(match); len(m) > 1 {
				aria = ` aria-describedby="` + m[1] + `"`
			}
			return fmt.Sprintf(`<textarea class="form-input admin-form-control admin-form-textarea" id="%s" name="%s" rows="4"%s>%s</textarea>`, scope, scope, aria, value)
		})
	}
	return formHTML
}

func textareaScopes(uiSchema json.RawMessage) []string {
	var node map[string]any
	if err := json.Unmarshal(uiSchema, &node); err != nil {
		return nil
	}
	var scopes []string
	collectTextareaScopes(node, &scopes)
	return scopes
}

func collectTextareaScopes(node map[string]any, scopes *[]string) {
	if node == nil {
		return
	}
	if nodeType, _ := node["type"].(string); nodeType == "Control" {
		if scope, _ := node["scope"].(string); scope != "" {
			if options, ok := node["options"].(map[string]any); ok {
				if multi, ok := options["multi"].(bool); ok && multi {
					*scopes = append(*scopes, scope)
				}
			}
		}
	}
	if elements, ok := node["elements"].([]any); ok {
		for _, el := range elements {
			if child, ok := el.(map[string]any); ok {
				collectTextareaScopes(child, scopes)
			}
		}
	}
}

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
		if len(parts) < 6 {
			return match
		}
		columnClass := parts[1]
		scope := parts[2]
		title := parts[3]
		smallHTML := ""
		if len(parts) > 7 {
			smallHTML = parts[7]
		}
		return renderBooleanToggle(scope, title, smallHTML, data, columnClass)
	})
}

func renderBooleanToggle(scope, title, smallHTML string, data map[string]any, columnClass string) string {
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
	return fmt.Sprintf(`<div class="form-group%s">
  <label class="form-label admin-form-label" for="%s">%s</label>
  <label class="admin-form-toggle admin-jsonforms-toggle">
    <input type="checkbox" class="admin-form-toggle-input" id="%s" name="%s" value="true"%s>
    <span class="admin-form-toggle-track" aria-hidden="true"><span class="admin-form-toggle-thumb"></span></span>
    <span class="admin-form-toggle-text">
      <span class="admin-form-toggle-label-on">Enabled</span>
      <span class="admin-form-toggle-label-off">Disabled</span>
    </span>
  </label>%s
</div>`, columnClass, scope, title, scope, scope, checkedAttr, smallBlock)
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
