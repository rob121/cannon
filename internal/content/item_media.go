package content

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/rob121/cannon/internal/models"
)

// GalleryImage is one image in an item gallery.
type GalleryImage struct {
	URL string `json:"url"`
	Alt string `json:"alt,omitempty"`
}

// ItemEmbed is embedded video/audio or external media.
type ItemEmbed struct {
	URL   string `json:"url"`
	Title string `json:"title,omitempty"`
	Kind  string `json:"kind,omitempty"` // video, audio, iframe
}

// ItemAttachment is a downloadable file linked to an item.
type ItemAttachment struct {
	URL   string `json:"url"`
	Label string `json:"label,omitempty"`
}

func ParseGalleryJSON(raw string) []GalleryImage {
	out, _ := parseJSONArray[GalleryImage](raw)
	clean := make([]GalleryImage, 0, len(out))
	for _, img := range out {
		img.URL = strings.TrimSpace(img.URL)
		if img.URL == "" {
			continue
		}
		clean = append(clean, img)
	}
	return clean
}

func ParseEmbedsJSON(raw string) []ItemEmbed {
	out, _ := parseJSONArray[ItemEmbed](raw)
	clean := make([]ItemEmbed, 0, len(out))
	for _, embed := range out {
		embed.URL = strings.TrimSpace(embed.URL)
		if embed.URL == "" {
			continue
		}
		if embed.Kind == "" {
			embed.Kind = "iframe"
		}
		clean = append(clean, embed)
	}
	return clean
}

func ParseAttachmentsJSON(raw string) []ItemAttachment {
	out, _ := parseJSONArray[ItemAttachment](raw)
	clean := make([]ItemAttachment, 0, len(out))
	for _, att := range out {
		att.URL = strings.TrimSpace(att.URL)
		if att.URL == "" {
			continue
		}
		if att.Label == "" {
			att.Label = filepathBase(att.URL)
		}
		clean = append(clean, att)
	}
	return clean
}

func EncodeGalleryJSON(items []GalleryImage) string {
	return encodeJSONArray(items)
}

func EncodeEmbedsJSON(items []ItemEmbed) string {
	return encodeJSONArray(items)
}

func EncodeAttachmentsJSON(items []ItemAttachment) string {
	return encodeJSONArray(items)
}

// ItemMediaFromForm builds JSON fields from structured admin form inputs.
func ItemMediaFromForm(r *http.Request) (gallery, embeds, attachments string) {
	gallery = EncodeGalleryJSON(galleryFromForm(r))
	embeds = EncodeEmbedsJSON(embedsFromForm(r))
	attachments = EncodeAttachmentsJSON(attachmentsFromForm(r))
	return gallery, embeds, attachments
}

func galleryFromForm(r *http.Request) []GalleryImage {
	urls := r.Form["gallery_url"]
	alts := r.Form["gallery_alt"]
	out := make([]GalleryImage, 0, len(urls))
	for i, url := range urls {
		url = strings.TrimSpace(url)
		if url == "" {
			continue
		}
		alt := ""
		if i < len(alts) {
			alt = strings.TrimSpace(alts[i])
		}
		out = append(out, GalleryImage{URL: url, Alt: alt})
	}
	return out
}

func embedsFromForm(r *http.Request) []ItemEmbed {
	urls := r.Form["embed_url"]
	titles := r.Form["embed_title"]
	kinds := r.Form["embed_kind"]
	out := make([]ItemEmbed, 0, len(urls))
	for i, url := range urls {
		url = strings.TrimSpace(url)
		if url == "" {
			continue
		}
		title := ""
		if i < len(titles) {
			title = strings.TrimSpace(titles[i])
		}
		kind := "iframe"
		if i < len(kinds) {
			kind = normalizeEmbedKind(kinds[i])
		}
		out = append(out, ItemEmbed{URL: url, Title: title, Kind: kind})
	}
	return out
}

func attachmentsFromForm(r *http.Request) []ItemAttachment {
	urls := r.Form["attachment_url"]
	labels := r.Form["attachment_label"]
	out := make([]ItemAttachment, 0, len(urls))
	for i, url := range urls {
		url = strings.TrimSpace(url)
		if url == "" {
			continue
		}
		label := ""
		if i < len(labels) {
			label = strings.TrimSpace(labels[i])
		}
		out = append(out, ItemAttachment{URL: url, Label: label})
	}
	return out
}

func normalizeEmbedKind(kind string) string {
	switch strings.ToLower(strings.TrimSpace(kind)) {
	case "video", "audio", "iframe":
		return strings.ToLower(strings.TrimSpace(kind))
	default:
		return "iframe"
	}
}

func parseJSONArray[T any](raw string) ([]T, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" || raw == "[]" {
		return nil, nil
	}
	var out []T
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		return nil, err
	}
	return out, nil
}

func encodeJSONArray(v any) string {
	b, err := json.Marshal(v)
	if err != nil {
		return "[]"
	}
	if string(b) == "null" {
		return "[]"
	}
	return string(b)
}

func filepathBase(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return "Download"
	}
	if idx := strings.LastIndexAny(path, "/\\"); idx >= 0 && idx < len(path)-1 {
		return path[idx+1:]
	}
	return path
}

// ValidateRequiredCustomFields ensures required custom fields have values.
func ValidateRequiredCustomFields(fields []models.ContentField, r *http.Request) error {
	return ValidateCustomFields(fields, r)
}

// CustomFieldFormValue reads one custom field value from an item form submission.
func CustomFieldFormValue(field models.ContentField, r *http.Request) string {
	return customFieldFormValue(r, field)
}

func customFieldFormValue(r *http.Request, field models.ContentField) string {
	key := fmt.Sprintf("field_%d", field.FieldID)
	switch field.Type {
	case "boolean":
		if formBoolStatic(r, key) {
			return "1"
		}
		return ""
	case "multi_select", "checkbox":
		vals := r.Form[key]
		parts := make([]string, 0, len(vals))
		for _, v := range vals {
			v = strings.TrimSpace(v)
			if v != "" {
				parts = append(parts, v)
			}
		}
		return strings.Join(parts, ",")
	default:
		return strings.TrimSpace(r.FormValue(key))
	}
}

func formBoolStatic(r *http.Request, key string) bool {
	v := strings.TrimSpace(r.FormValue(key))
	return v == "on" || v == "1" || v == "true" || v == "yes"
}
