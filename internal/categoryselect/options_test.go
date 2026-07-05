package categoryselect_test

import (
	"testing"

	"github.com/rob121/cannon/internal/categoryselect"
	"github.com/rob121/cannon/internal/models"
)

func TestFlattenCategoryOptions(t *testing.T) {
	parentID := uint(1)
	childID := uint(2)
	rows := []models.Category{
		{CategoryID: 1, Name: "News"},
		{CategoryID: 2, Name: "Local", ParentID: &parentID},
		{CategoryID: 3, Name: "Sports"},
		{CategoryID: 4, Name: "High School", ParentID: &childID},
	}
	opts := categoryselect.Flatten(rows)
	if len(opts) != 4 {
		t.Fatalf("len(opts) = %d, want 4", len(opts))
	}
	if opts[0].Name != "News" || opts[0].Depth != 0 || opts[0].Label != "News" {
		t.Fatalf("root: %+v", opts[0])
	}
	if opts[1].Name != "Local" || opts[1].Depth != 1 || opts[1].Label != "— Local" {
		t.Fatalf("child: %+v", opts[1])
	}
	if opts[2].Name != "High School" || opts[2].Depth != 2 || opts[2].Label != "— — High School" {
		t.Fatalf("grandchild: %+v", opts[2])
	}
	if opts[3].Name != "Sports" || opts[3].Depth != 0 {
		t.Fatalf("second root: %+v", opts[3])
	}
}

func TestCategoryParentOptionsExcludesDescendants(t *testing.T) {
	parentID := uint(1)
	childID := uint(2)
	rows := []models.Category{
		{CategoryID: 1, Name: "News"},
		{CategoryID: 2, Name: "Local", ParentID: &parentID},
		{CategoryID: 3, Name: "Sports"},
		{CategoryID: 4, Name: "High School", ParentID: &childID},
	}
	opts := categoryselect.ParentOptions(rows, 1)
	if len(opts) != 1 || opts[0].CategoryID != 3 {
		t.Fatalf("opts: %+v", opts)
	}
}
