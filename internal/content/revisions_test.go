package content_test

import (
	"encoding/json"
	"testing"

	"github.com/rob121/cannon/internal/content"
	"github.com/rob121/cannon/internal/models"
)

func TestCompareRevisionSnapshots(t *testing.T) {
	a := content.ItemRevisionSnapshot{Title: "Old", Body: "Body A", Status: models.ItemStatusDraft}
	b := content.ItemRevisionSnapshot{Title: "New", Body: "Body B", Status: models.ItemStatusPending}
	diffs := content.CompareRevisionSnapshots(a, b)
	if len(diffs) < 3 {
		t.Fatalf("expected multiple diffs, got %+v", diffs)
	}
}

func TestSnapshotItemRoundTrip(t *testing.T) {
	item := &models.Item{
		Title:  "Hello",
		Slug:   "hello",
		Status: models.ItemStatusPending,
		Body:   "World",
	}
	snap := content.SnapshotItem(item, []uint{1, 2}, []uint{3}, map[uint]string{4: "x"})
	raw, err := json.Marshal(snap)
	if err != nil {
		t.Fatal(err)
	}
	var decoded content.ItemRevisionSnapshot
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded.Title != "Hello" || decoded.Status != models.ItemStatusPending {
		t.Fatalf("snapshot mismatch: %+v", decoded)
	}
	if decoded.FieldValues[4] != "x" {
		t.Fatalf("field values: %+v", decoded.FieldValues)
	}
}
