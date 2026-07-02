package templatemgr

import "testing"

func TestRootGroups(t *testing.T) {
	groups, err := RootGroups(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if len(groups) < 2 {
		t.Fatalf("expected admin and default groups, got %#v", groups)
	}
	names := map[string]bool{}
	for _, g := range groups {
		names[g.Name] = true
		if g.Total == 0 {
			t.Fatalf("group %q has no templates", g.Name)
		}
	}
	if !names["admin"] || !names["default"] {
		t.Fatalf("missing expected groups: %#v", groups)
	}
}

func TestGroupTemplates(t *testing.T) {
	entries, err := GroupTemplates(t.TempDir(), "admin")
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) == 0 {
		t.Fatal("expected admin templates")
	}
	found := false
	for _, entry := range entries {
		if entry.Path == "admin/dashboard.html" {
			found = true
			if entry.Name != "dashboard" {
				t.Fatalf("unexpected name: %q", entry.Name)
			}
		}
	}
	if !found {
		t.Fatal("dashboard template not listed")
	}
}

func TestValidateGroup(t *testing.T) {
	if err := ValidateGroup("admin"); err != nil {
		t.Fatal(err)
	}
	if err := ValidateGroup("../admin"); err == nil {
		t.Fatal("expected invalid group error")
	}
}
