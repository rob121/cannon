package admin

import (
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/rob121/cannon/internal/models"
)

func TestFlipStatus(t *testing.T) {
	if flipStatus(models.StatusActive) != models.StatusInactive {
		t.Fatal("expected inactive")
	}
	if flipStatus(models.StatusInactive) != models.StatusActive {
		t.Fatal("expected active")
	}
}

func TestListRedirectQueryPreservesFilters(t *testing.T) {
	req := httptest.NewRequest("GET", "/admin/blocks?page=2&sort=name&dir=desc&space=footer", nil)
	got := listRedirectQuery(req)
	q, err := url.ParseQuery(strings.TrimPrefix(got, "?"))
	if err != nil {
		t.Fatal(err)
	}
	if q.Get("page") != "2" || q.Get("sort") != "name" || q.Get("dir") != "desc" || q.Get("space") != "footer" {
		t.Fatalf("listRedirectQuery: got %v", q)
	}
}

func TestListRedirectQueryEmpty(t *testing.T) {
	req := httptest.NewRequest("GET", "/admin/users", nil)
	if got := listRedirectQuery(req); got != "" {
		t.Fatalf("expected empty query, got %q", got)
	}
}

func TestListRedirectQueryExtraOnly(t *testing.T) {
	req := httptest.NewRequest("GET", "/admin/blocks?"+url.Values{"space": {"sidebar"}}.Encode(), nil)
	got := listRedirectQuery(req)
	if got != "?space=sidebar" {
		t.Fatalf("got %q", got)
	}
}
