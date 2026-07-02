package admin

import "testing"

func TestNormalizeMediaFolder(t *testing.T) {
	got, err := normalizeMediaFolder(" content/reports/ ")
	if err != nil {
		t.Fatal(err)
	}
	if got != "content/reports" {
		t.Fatalf("got %q", got)
	}
	if _, err := normalizeMediaFolder("../secret"); err == nil {
		t.Fatal("expected error for traversal")
	}
}

func TestJoinMediaFolder(t *testing.T) {
	got, err := joinMediaFolder("content", "Reports 2026")
	if err != nil {
		t.Fatal(err)
	}
	if got != "content/reports-2026" {
		t.Fatalf("got %q", got)
	}
}

func TestDirectMediaChildFolder(t *testing.T) {
	paths := []string{"content", "content/reports", "content/reports/2026", "images"}
	children := mediaChildFolderNav(paths, "content", nil)
	if len(children) != 1 || children[0].Name != "content/reports" {
		t.Fatalf("children = %#v", children)
	}
	rootChildren := mediaChildFolderNav(paths, "", nil)
	if len(rootChildren) != 2 {
		t.Fatalf("root children = %#v", rootChildren)
	}
}

func TestMediaFolderBreadcrumbs(t *testing.T) {
	crumbs := mediaFolderBreadcrumbs("content/reports")
	if len(crumbs) != 3 {
		t.Fatalf("crumbs = %#v", crumbs)
	}
	if crumbs[2]["Label"] != "reports" {
		t.Fatalf("last crumb = %#v", crumbs[2])
	}
}
