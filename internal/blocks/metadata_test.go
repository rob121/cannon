package blocks

import "testing"

func TestMetadataFromForm(t *testing.T) {
	raw, err := MetadataFromForm("html", " <p>Hi</p> ", "")
	if err != nil {
		t.Fatal(err)
	}
	meta, err := ParseMetadata(raw)
	if err != nil {
		t.Fatal(err)
	}
	if meta.Content != "<p>Hi</p>" {
		t.Fatalf("content: got %q", meta.Content)
	}

	raw, err = MetadataFromForm("extension", "ignored", "42")
	if err != nil {
		t.Fatal(err)
	}
	meta, err = ParseMetadata(raw)
	if err != nil {
		t.Fatal(err)
	}
	if meta.Content != "" {
		t.Fatalf("extension content should be empty, got %q", meta.Content)
	}
	if meta.FormID != 42 {
		t.Fatalf("form_id: got %d", meta.FormID)
	}
}

func TestMetadataFromFormValuesExtensionMultiselect(t *testing.T) {
	values := make(map[string][]string)
	values["block_data_list_ids"] = []string{"1", "2", "3"}
	raw, err := MetadataFromFormValues("extension", "", values)
	if err != nil {
		t.Fatal(err)
	}
	m, err := MetadataMap(raw)
	if err != nil {
		t.Fatal(err)
	}
	listIDs, ok := m["list_ids"].([]any)
	if !ok {
		t.Fatalf("list_ids: %#v", m["list_ids"])
	}
	if len(listIDs) != 3 {
		t.Fatalf("expected 3 ids, got %#v", listIDs)
	}
}

func TestMetadataMap(t *testing.T) {
	m, err := MetadataMap(`{"form_id":7,"content":"x"}`)
	if err != nil {
		t.Fatal(err)
	}
	if m["form_id"].(float64) != 7 {
		t.Fatalf("form_id: %#v", m["form_id"])
	}
}
