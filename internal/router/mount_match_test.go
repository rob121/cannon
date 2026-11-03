package router

import (
	"testing"

	"github.com/rob121/cannon/internal/models"
)

func TestMatchExtensionMount(t *testing.T) {
	route := models.Route{
		Type:            models.RouteTypeExtension,
		ExtensionPageID: "",
		Path:            "/jobs",
	}
	if _, _, ok := matchRoutePath(route, "/jobs/comm-coordinator", ""); !ok {
		t.Fatal("expected mount to match subpath")
	}
	if _, _, ok := matchRoutePath(route, "/jobs", ""); !ok {
		t.Fatal("expected mount to match index path")
	}
	if _, _, ok := matchRoutePath(route, "/jobs-other", ""); ok {
		t.Fatal("expected prefix mismatch")
	}
}
