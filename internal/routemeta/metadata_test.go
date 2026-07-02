package routemeta_test

import (
	"net/url"
	"testing"

	"github.com/rob121/cannon/internal/routemeta"
)

func TestMetadataFromFormControllerData(t *testing.T) {
	values := url.Values{
		"controller_data_item_slug":     {"hello-world"},
		"controller_data_category_slug": {"news/local"},
		"ignored":                       {"x"},
	}
	raw, err := routemeta.MetadataFromForm(values)
	if err != nil {
		t.Fatal(err)
	}
	if got := routemeta.MetadataString(raw, "item_slug"); got != "hello-world" {
		t.Fatalf("item_slug = %q", got)
	}
	if got := routemeta.MetadataString(raw, "category_slug"); got != "news/local" {
		t.Fatalf("category_slug = %q", got)
	}
	if got := routemeta.MetadataString(raw, "ignored"); got != "" {
		t.Fatalf("unexpected ignored key: %q", got)
	}
}
