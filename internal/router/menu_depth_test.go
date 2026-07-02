package router

import "testing"

func TestLimitMenuDepth(t *testing.T) {
	grandchild := map[string]any{"Name": "Level 3", "Href": "/l3"}
	child := map[string]any{
		"Name":     "Level 2",
		"Href":     "/l2",
		"Children": []map[string]any{grandchild},
	}
	root := map[string]any{
		"Name":     "Level 1",
		"Href":     "/l1",
		"Children": []map[string]any{child},
	}
	tree := LimitMenuDepth([]map[string]any{root}, 3)
	if len(tree) != 1 {
		t.Fatalf("roots = %d", len(tree))
	}
	l2, ok := tree[0]["Children"].([]map[string]any)
	if !ok || len(l2) != 1 {
		t.Fatalf("level 2 = %#v", tree[0]["Children"])
	}
	l3, ok := l2[0]["Children"].([]map[string]any)
	if !ok || len(l3) != 1 || l3[0]["Name"] != "Level 3" {
		t.Fatalf("level 3 = %#v", l2[0]["Children"])
	}
	if _, ok := l3[0]["Children"]; ok {
		t.Fatal("expected no level 4 children")
	}

	trimmed := LimitMenuDepth([]map[string]any{root}, 2)
	l2trim, ok := trimmed[0]["Children"].([]map[string]any)
	if !ok || len(l2trim) != 1 {
		t.Fatalf("trimmed level 2 = %#v", trimmed[0]["Children"])
	}
	if _, ok := l2trim[0]["Children"]; ok {
		t.Fatal("expected children removed at depth 2 cap")
	}
}
