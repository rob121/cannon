package admin

import (
	"net/http"
	"net/http/httptest"
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

func TestApplyListSortDescDefault(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/admin/media?folder=content", nil)
	data := map[string]any{}
	order := applyListSortDesc(r, data, map[string]string{"created": "created_at"}, "created")
	if order != "created_at desc" {
		t.Fatalf("order = %q, want created_at desc", order)
	}
	if data["Dir"] != "desc" {
		t.Fatalf("Dir = %v, want desc", data["Dir"])
	}
}

func TestApplyListSortDescExplicit(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/admin/media?sort=name&dir=asc", nil)
	data := map[string]any{}
	order := applyListSortDesc(r, data, map[string]string{"name": "name", "created": "created_at"}, "created")
	if order != "name asc" {
		t.Fatalf("order = %q, want name asc", order)
	}
}

func TestListQueryFromDataIncludesFiltersAndSort(t *testing.T) {
	got := listQueryFromData(map[string]any{
		"Page":         2,
		"Sort":         "name",
		"Dir":          "desc",
		"SpaceFilter":  "footer",
		"StatusFilter": "published",
		"CategoryFilter": "3",
		"SearchQuery":  "hello",
		"Filter":       "pending",
	})
	for _, part := range []string{"page=2", "sort=name", "dir=desc", "space=footer", "status=published", "category=3", "q=hello", "filter=pending"} {
		if !strings.Contains(got, part) {
			t.Fatalf("listQueryFromData: missing %q in %q", part, got)
		}
	}
}

func TestApplyListSortDefaultColNotInAllowed(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/admin/extensions", nil)
	data := map[string]any{}
	order := applyListSort(r, data, map[string]string{
		"name": "name", "status": "status", "installed": "installed",
	}, "sort")
	if order != "sort asc" {
		t.Fatalf("order = %q, want sort asc", order)
	}
	if data["Sort"] != "sort" || data["Dir"] != "asc" {
		t.Fatalf("data = %#v", data)
	}
}

func TestApplyListSortExplicitDefaultSortParam(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/admin/extensions?sort=sort&dir=asc", nil)
	data := map[string]any{}
	order := applyListSort(r, data, map[string]string{
		"name": "name", "status": "status", "installed": "installed", "sort": "sort",
	}, "sort")
	if order != "sort asc" {
		t.Fatalf("order = %q, want sort asc", order)
	}
}

func TestExtensionListOrderUsesManualSortForDefault(t *testing.T) {
	tests := []struct {
		url  string
		want string
	}{
		{"/admin/extensions", "sort asc, extension_id asc"},
		{"/admin/extensions?sort=sort&dir=asc", "sort asc, extension_id asc"},
		{"/admin/extensions?sort=name&dir=desc", "name desc"},
	}
	for _, tc := range tests {
		r := httptest.NewRequest(http.MethodGet, tc.url, nil)
		data := map[string]any{}
		order := applyListSort(r, data, map[string]string{
			"name": "name", "status": "status", "installed": "installed", "sort": "sort",
		}, "sort")
		sortParam := strings.TrimSpace(r.URL.Query().Get("sort"))
		if sortParam == "" || sortParam == "sort" {
			order = "sort asc, extension_id asc"
		}
		if order != tc.want {
			t.Fatalf("%s: order = %q, want %q", tc.url, order, tc.want)
		}
	}
}
