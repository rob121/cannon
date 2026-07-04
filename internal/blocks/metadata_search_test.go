package blocks

import (
	"net/url"
	"testing"
)

func TestMetadataFromFormValuesSearchBlock(t *testing.T) {
	values := url.Values{
		"search_placeholder": {"Find articles"},
		"search_button":      {"Go"},
		"search_label":         {"Site search"},
		"search_class":         {"sidebar-search"},
	}
	raw, err := MetadataFromFormValues("search-vertical", "", values)
	if err != nil {
		t.Fatal(err)
	}
	meta, err := ParseMetadata(raw)
	if err != nil {
		t.Fatal(err)
	}
	if meta.SearchPlaceholder != "Find articles" ||
		meta.SearchButton != "Go" ||
		meta.SearchLabel != "Site search" ||
		meta.SearchClass != "sidebar-search" {
		t.Fatalf("meta = %#v", meta)
	}
}
