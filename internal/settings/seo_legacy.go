package settings

import (
	"encoding/json"

	"github.com/rob121/cannon/extension"
)

var legacySEOMetaKeys = []string{
	"site_meta_description",
	"site_meta_keywords",
	"site_og_title",
	"site_og_image",
	"site_twitter_card",
	"site_twitter_site",
	"site_twitter_creator",
	"site_head_extra",
}

// MergeLegacySEOMetaSection copies default meta tag values from the general section
// when they have not yet been saved under SEO & Robots.
func MergeLegacySEOMetaSection(section extension.ConfigurationSection, generalData map[string]any) extension.ConfigurationSection {
	if len(generalData) == 0 {
		return section
	}
	seoData := map[string]any{}
	if len(section.Data) > 0 {
		_ = json.Unmarshal(section.Data, &seoData)
	}
	changed := false
	for _, key := range legacySEOMetaKeys {
		if hasStoredValue(seoData, key) {
			continue
		}
		if v, ok := generalData[key]; ok && hasStoredValue(map[string]any{key: v}, key) {
			seoData[key] = v
			changed = true
		}
	}
	if !changed {
		return section
	}
	raw, err := json.Marshal(seoData)
	if err != nil {
		return section
	}
	section.Data = raw
	return section
}

func hasStoredValue(data map[string]any, key string) bool {
	v, ok := data[key]
	if !ok || v == nil {
		return false
	}
	switch n := v.(type) {
	case string:
		return n != ""
	default:
		return true
	}
}
