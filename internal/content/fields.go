package content

import (
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"net/http"
	"strconv"
	"strings"

	"github.com/rob121/cannon/internal/models"
	"gorm.io/gorm"
)

// FieldOption is one selectable value from field configuration.
type FieldOption struct {
	Value string `json:"value"`
	Label string `json:"label"`
}

// FieldConfig is parsed ContentField.Configuration JSON.
type FieldConfig struct {
	Options []FieldOption `json:"options"`
	Min     *float64      `json:"min"`
	Max     *float64      `json:"max"`
	Pattern string        `json:"pattern"`
}

// FieldDisplay is a labeled custom field value for templates.
type FieldDisplay struct {
	Name  string
	Label string
	Type  string
	Value string
	HTML  string
}

// ParseFieldConfig unmarshals field configuration JSON.
func ParseFieldConfig(raw string) FieldConfig {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return FieldConfig{}
	}
	var cfg FieldConfig
	_ = json.Unmarshal([]byte(raw), &cfg)
	return cfg
}

// ValidateFieldValue checks a submitted value against field rules.
func ValidateFieldValue(field models.ContentField, value string) error {
	value = strings.TrimSpace(value)
	if field.Required && value == "" {
		label := strings.TrimSpace(field.Label)
		if label == "" {
			label = field.Name
		}
		return fmt.Errorf("%s is required", label)
	}
	if value == "" {
		return nil
	}
	cfg := ParseFieldConfig(field.Configuration)
	switch field.Type {
	case "number":
		n, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return fmt.Errorf("%s must be a number", field.Label)
		}
		if cfg.Min != nil && n < *cfg.Min {
			return fmt.Errorf("%s must be at least %g", field.Label, *cfg.Min)
		}
		if cfg.Max != nil && n > *cfg.Max {
			return fmt.Errorf("%s must be at most %g", field.Label, *cfg.Max)
		}
	case "url", "image", "file":
		if !strings.HasPrefix(value, "http://") && !strings.HasPrefix(value, "https://") && !strings.HasPrefix(value, "/") {
			return fmt.Errorf("%s must be a valid URL or path", field.Label)
		}
	case "select", "radio":
		if !fieldOptionAllowed(cfg, value) {
			return fmt.Errorf("%s has an invalid option", field.Label)
		}
	case "multi_select", "checkbox":
		for _, part := range splitMultiValue(value) {
			if part != "" && !fieldOptionAllowed(cfg, part) {
				return fmt.Errorf("%s has an invalid option", field.Label)
			}
		}
	}
	return nil
}

func fieldOptionAllowed(cfg FieldConfig, value string) bool {
	if len(cfg.Options) == 0 {
		return true
	}
	for _, opt := range cfg.Options {
		if opt.Value == value {
			return true
		}
	}
	return false
}

func splitMultiValue(value string) []string {
	parts := strings.Split(value, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}

// FormatFieldDisplayHTML renders a stored value for frontend output.
func FormatFieldDisplayHTML(field models.ContentField, value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	cfg := ParseFieldConfig(field.Configuration)
	switch field.Type {
	case "boolean":
		if value == "1" || strings.EqualFold(value, "true") || strings.EqualFold(value, "yes") {
			return "Yes"
		}
		return "No"
	case "url":
		safe := html.EscapeString(value)
		return `<a href="` + safe + `" rel="noopener">` + safe + `</a>`
	case "image":
		alt := html.EscapeString(field.Label)
		return `<img src="` + html.EscapeString(value) + `" alt="` + alt + `" class="img-fluid rounded">`
	case "file":
		label := html.EscapeString(value)
		if idx := strings.LastIndexAny(value, "/\\"); idx >= 0 && idx < len(value)-1 {
			label = html.EscapeString(value[idx+1:])
		}
		return `<a href="` + html.EscapeString(value) + `" download>` + label + `</a>`
	case "rich_text":
		out, err := RichTextToHTML(value)
		if err != nil {
			return html.EscapeString(value)
		}
		return out
	case "textarea":
		out, err := RichTextToHTML(value)
		if err != nil {
			return html.EscapeString(value)
		}
		return out
	case "multi_select", "checkbox":
		labels := make([]string, 0)
		for _, part := range splitMultiValue(value) {
			labels = append(labels, html.EscapeString(fieldOptionLabel(cfg, part)))
		}
		return strings.Join(labels, ", ")
	case "select", "radio":
		return html.EscapeString(fieldOptionLabel(cfg, value))
	default:
		return html.EscapeString(value)
	}
}

func fieldOptionLabel(cfg FieldConfig, value string) string {
	for _, opt := range cfg.Options {
		if opt.Value == value {
			if strings.TrimSpace(opt.Label) != "" {
				return opt.Label
			}
			return opt.Value
		}
	}
	return value
}

// ValidateCustomFields validates all fields for an item form submission.
func ValidateCustomFields(fields []models.ContentField, r *http.Request) error {
	for _, field := range fields {
		if err := ValidateFieldValue(field, CustomFieldFormValue(field, r)); err != nil {
			return err
		}
	}
	return nil
}

// FieldValueContains reports whether a comma-separated stored value includes part.
func FieldValueContains(value, part string) bool {
	part = strings.TrimSpace(part)
	if part == "" {
		return false
	}
	for _, v := range splitMultiValue(value) {
		if v == part {
			return true
		}
	}
	return false
}

// SaveItemFieldValues: saves custom field values from a form submission.
func SaveItemFieldValues(db *gorm.DB, itemID uint, fields []models.ContentField, r *http.Request) error {
	for _, field := range fields {
		value := CustomFieldFormValue(field, r)
		var existing models.ItemFieldValue
		err := db.Where("item_id = ? AND field_id = ?", itemID, field.FieldID).First(&existing).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			if value == "" {
				continue
			}
			if err := db.Create(&models.ItemFieldValue{ItemID: itemID, FieldID: field.FieldID, Value: value}).Error; err != nil {
				return err
			}
			continue
		}
		if err != nil {
			return err
		}
		if value == "" {
			if err := db.Delete(&existing).Error; err != nil {
				return err
			}
			continue
		}
		existing.Value = value
		if err := db.Save(&existing).Error; err != nil {
			return err
		}
	}
	return nil
}
