package router

import (
	"testing"
	"time"

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

func TestMenuItemPublishWindowHidesLikeInactive(t *testing.T) {
	now := time.Date(2026, 10, 25, 12, 0, 0, 0, time.UTC)
	future := now.Add(2 * time.Hour)
	past := now.Add(-2 * time.Hour)
	openStart := now.Add(-time.Hour)
	openEnd := now.Add(time.Hour)
	parentID := uint(1)
	public := []models.Group{{GroupID: 9, Name: "public", Kind: models.GroupKindFrontend, Status: models.StatusActive}}

	items := []models.MenuItem{
		{MenuItemID: 1, Name: "Scheduled", Status: models.StatusActive, PublishStart: &future, Groups: public},
		{MenuItemID: 2, Name: "Expired", Status: models.StatusActive, PublishEnd: &past, Groups: public},
		{MenuItemID: 3, Name: "Open", Status: models.StatusActive, PublishStart: &openStart, PublishEnd: &openEnd, Groups: public},
		{MenuItemID: 4, Name: "Always", Status: models.StatusActive, Groups: public},
		{MenuItemID: 5, Name: "Child", Status: models.StatusActive, ParentID: &parentID, Groups: public},
		{MenuItemID: 6, Name: "Off", Status: models.StatusInactive, Groups: public},
	}

	got := filterPublicMenuItems(items, []uint{9}, now)
	names := make([]string, 0, len(got))
	for _, item := range got {
		names = append(names, item.Name)
	}
	want := []string{"Open", "Always"}
	if len(names) != len(want) {
		t.Fatalf("visible = %v, want %v", names, want)
	}
	for i := range want {
		if names[i] != want[i] {
			t.Fatalf("visible = %v, want %v", names, want)
		}
	}
	if MenuItemVisibleAt(items[2], now) != true {
		t.Fatal("expected item inside the window to be visible")
	}
	if MenuItemVisibleAt(items[2], openEnd) != true {
		t.Fatal("expected item to stay visible at the publish end")
	}
	if MenuItemVisibleAt(items[2], openEnd.Add(time.Second)) {
		t.Fatal("expected item to hide just after the publish end")
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
