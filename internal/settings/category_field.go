package settings

import (
	"encoding/json"
	"fmt"
	"html"
	"net/url"
	"regexp"
	"strconv"
	"strings"

	"github.com/rob121/cannon/internal/models"
)

// RenderCategorySelectHTML builds a category dropdown for JSON Forms configuration fields.
func RenderCategorySelectHTML(scope string, categories []models.Category, selectedID uint) string {
	var b strings.Builder
	b.WriteString(`<select class="form-select admin-form-control admin-config-category" id="`)
	b.WriteString(html.EscapeString(scope))
	b.WriteString(`" name="`)
	b.WriteString(html.EscapeString(scope))
	b.WriteString(`"><option value="">— None —</option>`)
	seen := map[uint]bool{}
	for _, cat := range categories {
		seen[cat.CategoryID] = true
		selected := ""
		if cat.CategoryID == selectedID {
			selected = " selected"
		}
		b.WriteString(fmt.Sprintf(`<option value="%d"%s>%s</option>`, cat.CategoryID, selected, html.EscapeString(cat.Name)))
	}
	if selectedID > 0 && !seen[selectedID] {
		b.WriteString(fmt.Sprintf(`<option value="%d" selected>Category #%d</option>`, selectedID, selectedID))
	}
	b.WriteString(`</select>`)
	if len(categories) == 0 && selectedID == 0 {
		b.WriteString(`<small class="form-text text-muted">No categories defined yet. Add one under <strong>Categories</strong> in the admin.</small>`)
	}
	return b.String()
}

func normalizeCategoryFields(formHTML string, schema, uiSchema json.RawMessage, data map[string]any, categories []models.Category) string {
	fields := categoryFields(schema, uiSchema)
	for _, field := range fields {
		selected := categoryFieldSelectedID(data, field.Key)
		selectHTML := RenderCategorySelectHTML(field.Scope, categories, selected)
		scope := regexp.QuoteMeta(field.Scope)
		inputRE := regexp.MustCompile(`(?s)<input class="form-input admin-form-control" id="` + scope + `" name="` + scope + `"([^>]*)/>`)
		if inputRE.MatchString(formHTML) {
			formHTML = inputRE.ReplaceAllString(formHTML, selectHTML)
			continue
		}
		selectRE := regexp.MustCompile(`(?s)<select class="form-select admin-form-control" id="` + scope + `" name="` + scope + `"[^>]*>.*?</select>`)
		formHTML = selectRE.ReplaceAllString(formHTML, selectHTML)
	}
	return formHTML
}

func categoryFieldSelectedID(data map[string]any, key string) uint {
	if data == nil {
		return 0
	}
	v, ok := data[key]
	if !ok || v == nil {
		return 0
	}
	switch n := v.(type) {
	case float64:
		if n > 0 {
			return uint(n)
		}
	case int:
		if n > 0 {
			return uint(n)
		}
	case int64:
		if n > 0 {
			return uint(n)
		}
	case uint:
		return n
	case uint64:
		return uint(n)
	case string:
		id, err := strconv.ParseUint(strings.TrimSpace(n), 10, 64)
		if err == nil && id > 0 {
			return uint(id)
		}
	}
	return 0
}

// ApplyCategoryFormDefaults normalizes posted category dropdown values.
func ApplyCategoryFormDefaults(schema, uiSchema json.RawMessage, data map[string]any, form url.Values) map[string]any {
	if data == nil {
		data = map[string]any{}
	}
	for _, field := range categoryFields(schema, uiSchema) {
		formKey := field.Scope
		if !form.Has(formKey) {
			continue
		}
		raw := strings.TrimSpace(form.Get(formKey))
		if raw == "" {
			if propertyAllowsNull(schema, field.Key) {
				data[field.Key] = nil
			} else {
				data[field.Key] = 0
			}
			continue
		}
		id, err := strconv.ParseUint(raw, 10, 64)
		if err != nil || id == 0 {
			if propertyAllowsNull(schema, field.Key) {
				data[field.Key] = nil
			} else {
				data[field.Key] = 0
			}
			continue
		}
		data[field.Key] = id
	}
	return data
}
