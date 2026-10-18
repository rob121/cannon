package notifications

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"

	"github.com/rob121/cannon/internal/models"
)

// EventTemplate is a Layer 1 per-event subject/body override.
type EventTemplate struct {
	Subject string `json:"subject,omitempty"`
	Body    string `json:"body,omitempty"`
}

// ParseTemplatesJSON decodes notification templates map.
func ParseTemplatesJSON(raw string) map[string]EventTemplate {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return map[string]EventTemplate{}
	}
	var out map[string]EventTemplate
	if err := json.Unmarshal([]byte(raw), &out); err != nil || out == nil {
		return map[string]EventTemplate{}
	}
	return out
}

// TemplatesToJSON encodes templates for storage.
func TemplatesToJSON(templates map[string]EventTemplate) string {
	if len(templates) == 0 {
		return ""
	}
	clean := make(map[string]EventTemplate, len(templates))
	for event, tmpl := range templates {
		event = strings.TrimSpace(event)
		tmpl.Subject = strings.TrimSpace(tmpl.Subject)
		tmpl.Body = strings.TrimSpace(tmpl.Body)
		if event == "" || (tmpl.Subject == "" && tmpl.Body == "") {
			continue
		}
		clean[event] = tmpl
	}
	if len(clean) == 0 {
		return ""
	}
	raw, err := json.Marshal(clean)
	if err != nil {
		return ""
	}
	return string(raw)
}

// RenderEventMessage builds Layer 1 subject/body for an event.
func RenderEventMessage(templatesJSON, event string, args map[string]any) (subject, body string) {
	templates := ParseTemplatesJSON(templatesJSON)
	tmpl := templates[event]
	vars := FlattenArgs(args)
	if strings.TrimSpace(tmpl.Body) != "" {
		body = applyTemplate(tmpl.Body, vars)
	} else {
		body = defaultMessageBody(event, vars)
	}
	if strings.TrimSpace(tmpl.Subject) != "" {
		subject = applyTemplate(tmpl.Subject, vars)
	}
	return subject, body
}

func defaultMessageBody(event string, vars map[string]string) string {
	parts := []string{"Cannon: " + event}
	for _, key := range []string{"username", "email", "user_id", "item_id", "comment_id", "status", "title"} {
		if v := vars[key]; v != "" {
			parts = append(parts, fmt.Sprintf("%s=%s", key, v))
		}
	}
	return strings.Join(parts, " ")
}

func applyTemplate(raw string, vars map[string]string) string {
	out := raw
	for key, value := range vars {
		out = strings.ReplaceAll(out, "{{"+key+"}}", value)
	}
	return out
}

// FlattenArgs turns hook arguments into simple placeholder values.
func FlattenArgs(args map[string]any) map[string]string {
	out := map[string]string{}
	if args == nil {
		return out
	}
	for key, value := range args {
		key = strings.TrimSpace(key)
		if key == "" || value == nil {
			continue
		}
		switch key {
		case "item":
			flattenItem(out, value)
		case "comment":
			flattenComment(out, value)
		default:
			if s := stringifyArg(value); s != "" {
				out[key] = s
			}
		}
	}
	return out
}

func flattenItem(out map[string]string, value any) {
	switch item := value.(type) {
	case models.Item:
		setItemFields(out, &item)
	case *models.Item:
		if item != nil {
			setItemFields(out, item)
		}
	default:
		rv := reflect.ValueOf(value)
		if rv.Kind() == reflect.Pointer && !rv.IsNil() {
			rv = rv.Elem()
		}
		if rv.IsValid() && rv.Kind() == reflect.Struct {
			if item, ok := rv.Interface().(models.Item); ok {
				setItemFields(out, &item)
			}
		}
	}
}

func setItemFields(out map[string]string, item *models.Item) {
	if item == nil {
		return
	}
	out["item_id"] = fmt.Sprint(item.ItemID)
	out["title"] = item.Title
	out["slug"] = item.Slug
	out["status"] = string(item.Status)
	if item.AuthorID != nil {
		out["author_id"] = fmt.Sprint(*item.AuthorID)
	}
	if item.CategoryID != nil {
		out["category_id"] = fmt.Sprint(*item.CategoryID)
	}
	if item.Category != nil {
		out["category_slug"] = item.Category.Slug
		out["category"] = item.Category.Slug
	}
}

func flattenComment(out map[string]string, value any) {
	switch comment := value.(type) {
	case models.Comment:
		out["comment_id"] = fmt.Sprint(comment.CommentID)
		out["item_id"] = fmt.Sprint(comment.ItemID)
		if comment.UserID != nil {
			out["comment_user_id"] = fmt.Sprint(*comment.UserID)
		}
	case *models.Comment:
		if comment != nil {
			flattenComment(out, *comment)
		}
	}
}

func stringifyArg(value any) string {
	switch v := value.(type) {
	case string:
		return strings.TrimSpace(v)
	case fmt.Stringer:
		return strings.TrimSpace(v.String())
	default:
		rv := reflect.ValueOf(value)
		if !rv.IsValid() {
			return ""
		}
		if rv.Kind() == reflect.Pointer {
			if rv.IsNil() {
				return ""
			}
			return stringifyArg(rv.Elem().Interface())
		}
		switch rv.Kind() {
		case reflect.Struct, reflect.Map, reflect.Slice, reflect.Array, reflect.Func, reflect.Chan:
			return ""
		default:
			s := strings.TrimSpace(fmt.Sprint(v))
			if s == "<nil>" {
				return ""
			}
			return s
		}
	}
}
