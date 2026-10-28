package admin

import "testing"

func TestBlockMetaSelected(t *testing.T) {
	meta := map[string]any{
		"list_ids": []any{float64(1), float64(2)},
		"form_id":  float64(3),
	}
	if !blockMetaSelected(meta, "list_ids", "1") || !blockMetaSelected(meta, "list_ids", "2") {
		t.Fatal("expected multiselect values to match")
	}
	if blockMetaSelected(meta, "list_ids", "9") {
		t.Fatal("unexpected match")
	}
	if !blockMetaSelected(meta, "form_id", "3") {
		t.Fatal("expected scalar match")
	}
}
