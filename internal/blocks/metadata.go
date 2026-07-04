package blocks

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"
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
	RouteMode        string     `json:"route_mode,omitempty"`
	RouteIDs         []uint     `json:"route_ids,omitempty"`
	TemplateWrapper  string     `json:"template_wrapper,omitempty"`
	ShowName         bool       `json:"show_name,omitempty"`
	PublishStart     *time.Time `json:"publish_start,omitempty"`
	PublishEnd       *time.Time `json:"publish_end,omitempty"`
	LoginTitle       string     `json:"login_title,omitempty"`
	MenuName          string     `json:"menu_name,omitempty"`
	MenuClass         string     `json:"menu_class,omitempty"`
	SearchPlaceholder string     `json:"search_placeholder,omitempty"`
	SearchButton      string     `json:"search_button,omitempty"`
	SearchLabel       string     `json:"search_label,omitempty"`
	SearchClass       string     `json:"search_class,omitempty"`
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
	case "login":
		meta["login_title"] = strings.TrimSpace(values.Get("login_title"))
	case "menu-vertical", "menu-horizontal":
		meta["menu_name"] = strings.TrimSpace(values.Get("menu_name"))
		meta["menu_class"] = strings.TrimSpace(values.Get("menu_class"))
	case "search-horizontal", "search-vertical":
		meta["search_placeholder"] = strings.TrimSpace(values.Get("search_placeholder"))
		meta["search_button"] = strings.TrimSpace(values.Get("search_button"))
		meta["search_label"] = strings.TrimSpace(values.Get("search_label"))
		meta["search_class"] = strings.TrimSpace(values.Get("search_class"))
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
	applyCommonMeta(meta, values)
	applyRouteMeta(meta, values)
	raw, err := json.Marshal(meta)
	if err != nil {
		return "", err
	}
	return string(raw), nil
}

func applyCommonMeta(meta map[string]any, values url.Values) {
	wrapper := strings.TrimSpace(values.Get("template_wrapper"))
	if wrapper != "" {
		meta["template_wrapper"] = wrapper
	} else {
		delete(meta, "template_wrapper")
	}
	if formBool(values.Get("show_name")) {
		meta["show_name"] = true
	} else {
		delete(meta, "show_name")
	}
	if t := parseFormTime(values.Get("publish_start")); t != nil {
		meta["publish_start"] = t.UTC().Format(time.RFC3339)
	} else {
		delete(meta, "publish_start")
	}
	if t := parseFormTime(values.Get("publish_end")); t != nil {
		meta["publish_end"] = t.UTC().Format(time.RFC3339)
	} else {
		delete(meta, "publish_end")
	}
}

func formBool(raw string) bool {
	switch strings.TrimSpace(strings.ToLower(raw)) {
	case "on", "1", "true", "yes":
		return true
	default:
		return false
	}
}

func parseFormTime(raw string) *time.Time {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	for _, layout := range []string{"2006-01-02T15:04", "2006-01-02 15:04", "2006-01-02", time.RFC3339} {
		if t, err := time.Parse(layout, raw); err == nil {
			return &t
		}
	}
	return nil
}

func applyRouteMeta(meta map[string]any, values url.Values) {
	mode := strings.TrimSpace(values.Get("route_mode"))
	if mode == "" {
		mode = RouteModeAll
	}
	meta["route_mode"] = mode
	ids := parseRouteIDsFromForm(values)
	if len(ids) > 0 {
		meta["route_ids"] = ids
	} else {
		delete(meta, "route_ids")
	}
}

func parseRouteIDsFromForm(values url.Values) []uint {
	raw := values["route_ids"]
	if len(raw) == 0 {
		return nil
	}
	ids := make([]uint, 0, len(raw))
	seen := map[uint]struct{}{}
	for _, item := range raw {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		n, err := strconv.ParseUint(item, 10, 64)
		if err != nil || n == 0 {
			continue
		}
		id := uint(n)
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		ids = append(ids, id)
	}
	return ids
}
