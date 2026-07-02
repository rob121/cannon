package routemeta

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

// MetadataFromForm builds route metadata JSON from admin form fields.
func MetadataFromForm(values url.Values) (string, error) {
	meta := map[string]any{}
	for key, vals := range values {
		if len(vals) == 0 {
			continue
		}
		var name string
		switch {
		case strings.HasPrefix(key, "page_data_"):
			name = strings.TrimPrefix(key, "page_data_")
		case strings.HasPrefix(key, "endpoint_data_"):
			name = strings.TrimPrefix(key, "endpoint_data_")
		case strings.HasPrefix(key, "controller_data_"):
			name = strings.TrimPrefix(key, "controller_data_")
		default:
			continue
		}
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
	raw, err := json.Marshal(meta)
	if err != nil {
		return "", err
	}
	return string(raw), nil
}

// SetMetadataString stores or removes one string metadata value.
func SetMetadataString(raw, key, value string) (string, error) {
	meta, err := MetadataMap(raw)
	if err != nil {
		return "", err
	}
	value = strings.TrimSpace(value)
	if value == "" {
		delete(meta, key)
	} else {
		meta[key] = value
	}
	out, err := json.Marshal(meta)
	if err != nil {
		return "", err
	}
	return string(out), nil
}

// MetadataMap returns metadata as a map for extension wire requests.
func MetadataMap(raw string) (map[string]any, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return map[string]any{}, nil
	}
	var out map[string]any
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		return nil, fmt.Errorf("decode route metadata: %w", err)
	}
	if out == nil {
		return map[string]any{}, nil
	}
	return out, nil
}

// MetadataString returns one string metadata value from route JSON.
func MetadataString(raw, key string) string {
	meta, err := MetadataMap(raw)
	if err != nil {
		return ""
	}
	value, ok := meta[key]
	if !ok || value == nil {
		return ""
	}
	switch typed := value.(type) {
	case string:
		return strings.TrimSpace(typed)
	case float64:
		if typed == float64(int64(typed)) {
			return strconv.FormatInt(int64(typed), 10)
		}
		return strconv.FormatFloat(typed, 'f', -1, 64)
	default:
		return strings.TrimSpace(fmt.Sprint(typed))
	}
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
