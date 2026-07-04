package content_test

import (
	"context"
	"testing"

	"github.com/rob121/cannon/internal/content"
)

func TestStripLocalePrefix(t *testing.T) {
	locales := []string{"en-US", "fr-FR"}
	stripped, locale, ok := content.StripLocalePrefix("/fr-FR/content/item/slug", locales, "en-US")
	if !ok || locale != "fr-FR" || stripped != "/content/item/slug" {
		t.Fatalf("got stripped=%q locale=%q ok=%v", stripped, locale, ok)
	}
	stripped, locale, ok = content.StripLocalePrefix("/content/item/slug", locales, "en-US")
	if ok || locale != "en-US" || stripped != "/content/item/slug" {
		t.Fatalf("unexpected default-path result stripped=%q locale=%q ok=%v", stripped, locale, ok)
	}
}

func TestLocalizedPath(t *testing.T) {
	ctx := content.WithLocale(context.Background(), "fr-FR")
	if got := content.LocalizedPath(ctx, "/content/search"); got != "/content/search" {
		t.Fatalf("without multilingual settings got %q", got)
	}
}

func TestParseFieldFilters(t *testing.T) {
	filters := content.ParseFieldFilters(map[string][]string{
		"q":         {"hello"},
		"cf_color":  {"blue"},
		"cf_size":   {""},
		"category":  {"1"},
	})
	if len(filters) != 1 || filters["color"] != "blue" {
		t.Fatalf("filters = %#v", filters)
	}
}

func TestFTSMatchQuery(t *testing.T) {
	// ftsMatchQuery is unexported; covered indirectly via search index tests.
}
