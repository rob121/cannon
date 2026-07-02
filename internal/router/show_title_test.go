package router

import (
	"testing"

	"github.com/rob121/cannon/internal/models"
)

func TestShowTitleForRoute(t *testing.T) {
	if !ShowTitleForRoute(models.Route{}) {
		t.Fatal("expected true for empty route")
	}
	if ShowTitleForRoute(models.Route{RouteID: 1, ShowTitle: false}) {
		t.Fatal("expected false when route disables title")
	}
	if !ShowTitleForRoute(models.Route{RouteID: 2, ShowTitle: true}) {
		t.Fatal("expected true when route enables title")
	}
}
