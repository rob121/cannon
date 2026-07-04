package lang

import (
	"html/template"
	"net/http"
)

const translationPreviewQueryParam = "tp"

// TranslationPreviewActive reports whether lang should append keys in parentheses.
// Enabled when tp=1 is present in the request URL.
func TranslationPreviewActive(r *http.Request) bool {
	return r != nil && r.URL.Query().Get(translationPreviewQueryParam) == "1"
}

// FuncMap returns template helpers for the given language manager.
func FuncMap(mgr *Manager, preview bool) template.FuncMap {
	return template.FuncMap{
		"lang": func(key string, pairs ...string) string {
			return formatLang(mgr, key, preview, pairs...)
		},
		"localeTag": func() string {
			if mgr != nil {
				return mgr.LocaleTag()
			}
			return "en"
		},
	}
}

func formatLang(mgr *Manager, key string, preview bool, pairs ...string) string {
	filtered, callPreview := filterTranslationPreviewPairs(pairs, preview)

	var text string
	if mgr != nil {
		text = mgr.Fmt(key, filtered...)
	} else {
		text = key
	}

	if callPreview && text != key {
		return text + " (" + key + ")"
	}
	return text
}

func filterTranslationPreviewPairs(pairs []string, preview bool) ([]string, bool) {
	if len(pairs) == 0 {
		return pairs, preview
	}

	callPreview := preview
	filtered := make([]string, 0, len(pairs))
	for i := 0; i+1 < len(pairs); i += 2 {
		if pairs[i] == translationPreviewQueryParam {
			if pairs[i+1] == "1" {
				callPreview = true
			}
			continue
		}
		filtered = append(filtered, pairs[i], pairs[i+1])
	}
	return filtered, callPreview
}

// TestFuncMap returns template helpers backed by embedded defaults.
func TestFuncMap() template.FuncMap {
	mgr, err := NewEmbeddedManager("en-US")
	if err != nil {
		return FuncMap(nil, false)
	}
	return FuncMap(mgr, false)
}
