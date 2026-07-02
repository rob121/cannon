package blocks

import (
	"net/url"
	"testing"
)

func TestMetadataFromFormValuesMenuBlock(t *testing.T) {
	values := url.Values{
		"menu_name":  {"footer"},
		"menu_class": {"nav-pills flex-column"},
	}
	raw, err := MetadataFromFormValues("menu-vertical", "", values)
	if err != nil {
		t.Fatal(err)
	}
	meta, err := ParseMetadata(raw)
	if err != nil {
		t.Fatal(err)
	}
	if meta.MenuName != "footer" || meta.MenuClass != "nav-pills flex-column" {
		t.Fatalf("meta = %#v", meta)
	}
}
