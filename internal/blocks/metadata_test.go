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

func TestMetadataMap(t *testing.T) {
	m, err := MetadataMap(`{"form_id":7,"content":"x"}`)
	if err != nil {
		t.Fatal(err)
	}
	if m["form_id"].(float64) != 7 {
		t.Fatalf("form_id: %#v", m["form_id"])
	}
}
