package admin

import (
	"strings"
	"testing"
)

func TestListQueryAmp(t *testing.T) {
	got := listQueryAmp(1, "name", "asc")
	if !strings.HasPrefix(got, "&") || !strings.Contains(got, "sort=name") || !strings.Contains(got, "dir=asc") {
		t.Fatalf("listQueryAmp: got %q", got)
	}
	if got := listQuery(1, "name", "asc"); !strings.HasPrefix(got, "?") {
		t.Fatalf("listQuery: got %q", got)
	}
}
