package admin

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/rob121/cannon/internal/models"
)

func TestRouteMatchesListFilters(t *testing.T) {
	row := models.Route{
		Name:      "Home",
		Path:      "/",
		Type:      models.RouteTypeController,
		Status:    models.StatusActive,
		IsDefault: true,
		ShowTitle: true,
		Target:    "content/index",
	}

	if !routeMatchesListFilters(row, "active", "", "", "", "") {
		t.Fatal("expected active route to match empty filters")
	}
	if routeMatchesListFilters(row, "inactive", "", "", "", "") {
		t.Fatal("expected status mismatch to exclude row")
	}
	if !routeMatchesListFilters(row, "", string(models.RouteTypeController), "", "", "") {
		t.Fatal("expected type match")
	}
	if routeMatchesListFilters(row, "", string(models.RouteTypeURL), "", "", "") {
		t.Fatal("expected type mismatch")
	}
	if !routeMatchesListFilters(row, "", "", "1", "", "") {
		t.Fatal("expected default filter match")
	}
	if routeMatchesListFilters(row, "", "", "0", "", "") {
		t.Fatal("expected default filter exclude")
	}
	if !routeMatchesListFilters(row, "", "", "", "1", "") {
		t.Fatal("expected show title match")
	}
	if !routeMatchesListFilters(row, "", "", "", "", "content") {
		t.Fatal("expected search in target")
	}
	if routeMatchesListFilters(row, "", "", "", "", "missing") {
		t.Fatal("expected search miss")
	}
}

func TestRouteListFiltersFromRequestRejectsUnknownValues(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/admin/routes?status=deleted&route_type=Unknown&default=2&show_title=maybe&q=+hello+", nil)
	status, routeType, defaultFilter, showTitle, query := routeListFiltersFromRequest(r)
	if status != "" || routeType != "" || defaultFilter != "" || showTitle != "" {
		t.Fatalf("unexpected filters: %q %q %q %q", status, routeType, defaultFilter, showTitle)
	}
	if query != "hello" {
		t.Fatalf("query = %q, want hello", query)
	}
}
