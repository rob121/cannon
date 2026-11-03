package router

import (
	"testing"

	"github.com/rob121/cannon/internal/models"
)

func TestMatchRoutePathWithQuery(t *testing.T) {
	route := models.Route{
		Type: models.RouteTypeURL,
		Path: "/event-registration?id=14",
	}
	if _, _, ok := matchRoutePath(route, "/event-registration", "id=14"); !ok {
		t.Fatal("expected query route to match")
	}
	if _, _, ok := matchRoutePath(route, "/event-registration", "id=15"); ok {
		t.Fatal("expected different query to miss")
	}
	if _, _, ok := matchRoutePath(route, "/event-registration", ""); ok {
		t.Fatal("expected missing query to miss")
	}
}

func TestQueryRouteWinsOverPathOnly(t *testing.T) {
	specific := models.Route{Type: models.RouteTypeURL, Path: "/event-registration?id=14"}
	generic := models.Route{Type: models.RouteTypeURL, Path: "/event-registration"}

	_, scoreSpecific, ok := matchRoutePath(specific, "/event-registration", "id=14")
	if !ok {
		t.Fatal("expected specific match")
	}
	_, scoreGeneric, ok := matchRoutePath(generic, "/event-registration", "id=14")
	if !ok {
		t.Fatal("expected generic path match")
	}
	if scoreSpecific <= scoreGeneric {
		t.Fatalf("specific=%d generic=%d", scoreSpecific, scoreGeneric)
	}
}

func TestQueriesMatchExactOrderIndependent(t *testing.T) {
	if !queriesMatchExact("a=1&b=2", "b=2&a=1") {
		t.Fatal("expected equivalent query keys to match")
	}
}
