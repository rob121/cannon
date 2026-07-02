package router

import (
	"testing"

	"github.com/rob121/cannon/internal/models"
)

func TestBuildMenuTreeNested(t *testing.T) {
	parentID := uint(1)
	childID := uint(2)
	items := []models.MenuItem{
		{MenuItemID: 1, Name: "About", Sort: 1},
		{MenuItemID: 2, Name: "Team", Sort: 1, ParentID: &parentID},
		{MenuItemID: 3, Name: "Home", Sort: 0},
	}
	tree := buildMenuTree(items)
	if len(tree) != 2 {
		t.Fatalf("roots = %d, want 2", len(tree))
	}
	if tree[0]["Name"] != "Home" {
		t.Fatalf("first root = %v", tree[0]["Name"])
	}
	if tree[1]["Name"] != "About" {
		t.Fatalf("second root = %v", tree[1]["Name"])
	}
	children, ok := tree[1]["Children"].([]map[string]any)
	if !ok || len(children) != 1 {
		t.Fatalf("About children = %#v", tree[1]["Children"])
	}
	if children[0]["Name"] != "Team" {
		t.Fatalf("child name = %v", children[0]["Name"])
	}
	if children[0]["MenuItemID"] != childID {
		t.Fatalf("child id = %v", children[0]["MenuItemID"])
	}
}

func TestMenuItemParentOptionsExcludesDescendants(t *testing.T) {
	parentID := uint(1)
	items := []models.MenuItem{
		{MenuItemID: 1, MenuID: 5, Name: "About"},
		{MenuItemID: 2, MenuID: 5, Name: "Team", ParentID: &parentID},
	}
	opts := MenuItemParentOptions(items, 5, 1)
	if len(opts) != 0 {
		t.Fatalf("expected no parent options when excluding About, got %+v", opts)
	}
	opts = MenuItemParentOptions(items, 5, 2)
	if len(opts) != 1 || opts[0].MenuItemID != 1 {
		t.Fatalf("expected About as parent option, got %+v", opts)
	}
}

func TestValidateMenuItemParentRejectsCycle(t *testing.T) {
	parentID := uint(1)
	items := []models.MenuItem{
		{MenuItemID: 1, MenuID: 5, Name: "About"},
		{MenuItemID: 2, MenuID: 5, Name: "Team", ParentID: &parentID},
	}
	row := items[0]
	child := parentID
	if err := ValidateMenuItemParent(items, row, &child); err == nil {
		t.Fatal("expected cycle validation error")
	}
}
