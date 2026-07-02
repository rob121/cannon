package blocks

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

// Metadata holds type-specific block configuration stored as JSON.
type Metadata struct {
	Content      string `json:"content,omitempty"`
	FormID       int    `json:"form_id,omitempty"`
	ContentMode  string `json:"content_mode,omitempty"`
	Limit        int    `json:"limit,omitempty"`
	CategorySlug string `json:"category_slug,omitempty"`
	TagSlug      string `json:"tag_slug,omitempty"`
	AuthorKey    string `json:"author_key,omitempty"`
	ItemSlug     string `json:"item_slug,omitempty"`
	Layout       string `json:"layout,omitempty"`
}

// ParseMetadata decodes block metadata JSON.
func ParseMetadata(raw string) (Metadata, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return Metadata{}, nil
	}
	var meta Metadata
	if err := json.Unmarshal([]byte(raw), &meta); err != nil {
		return Metadata{}, fmt.Errorf("decode block metadata: %w", err)
	}
	return meta, nil
}

// EncodeMetadata serializes block metadata.
func EncodeMetadata(meta Metadata) (string, error) {
	raw, err := json.Marshal(meta)
	if err != nil {
		return "", err
	}
	return string(raw), nil
}

// MetadataMap returns metadata as a map for extension wire requests.
func MetadataMap(raw string) (map[string]any, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return map[string]any{}, nil
	}
	var out map[string]any
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		return nil, fmt.Errorf("decode block metadata: %w", err)
	}
	if out == nil {
		return map[string]any{}, nil
	}
	return out, nil
}

// MetadataStringMap returns metadata as display-safe string values for admin forms.
func MetadataStringMap(raw string) map[string]string {
	values := map[string]string{}
	meta, err := MetadataMap(raw)
	if err != nil {
		return values
	}
	for key, value := range meta {
		switch typed := value.(type) {
		case string:
			values[key] = typed
		case float64:
			if typed == float64(int64(typed)) {
				values[key] = strconv.FormatInt(int64(typed), 10)
			} else {
				values[key] = strconv.FormatFloat(typed, 'f', -1, 64)
			}
		case nil:
			values[key] = ""
		default:
			values[key] = fmt.Sprint(typed)
		}
	}
	return values
}

// MetadataFromForm builds metadata from admin form fields.
func MetadataFromForm(blockType, content, formID string) (string, error) {
	values := url.Values{}
	if strings.TrimSpace(formID) != "" {
		values.Set("form_id", formID)
		values.Set("block_data_form_id", formID)
	}
	return MetadataFromFormValues(blockType, content, values)
}

// MetadataFromFormValues builds metadata from admin form fields.
func MetadataFromFormValues(blockType, content string, values url.Values) (string, error) {
	meta := map[string]any{}
	switch blockType {
	case "html", "markdown":
		meta["content"] = strings.TrimSpace(content)
	case "content":
		meta["content_mode"] = strings.TrimSpace(values.Get("content_mode"))
		if limit, err := strconv.Atoi(strings.TrimSpace(values.Get("content_limit"))); err == nil && limit > 0 {
			meta["limit"] = limit
		}
		meta["category_slug"] = strings.TrimSpace(values.Get("category_slug"))
		meta["tag_slug"] = strings.TrimSpace(values.Get("tag_slug"))
		meta["author_key"] = strings.TrimSpace(values.Get("author_key"))
		meta["item_slug"] = strings.TrimSpace(values.Get("item_slug"))
		meta["layout"] = strings.TrimSpace(values.Get("content_layout"))
	case "extension":
		for key, vals := range values {
			if !strings.HasPrefix(key, "block_data_") || len(vals) == 0 {
				continue
			}
			name := strings.TrimPrefix(key, "block_data_")
			if name == "" {
				continue
			}
			value := strings.TrimSpace(vals[0])
			if value == "" {
				continue
			}
			if id, err := strconv.Atoi(value); err == nil {
				meta[name] = id
			} else {
				meta[name] = value
			}
		}
	}
	if id, err := strconv.Atoi(strings.TrimSpace(values.Get("form_id"))); err == nil && id > 0 {
		meta["form_id"] = id
	}
	raw, err := json.Marshal(meta)
	if err != nil {
		return "", err
	}
	return string(raw), nil
}
