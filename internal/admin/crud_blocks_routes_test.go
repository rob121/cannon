package admin

import (
	"testing"

	"github.com/rob121/cannon/internal/models"
)

func TestBuildBlockRouteSectionsIncludesDefaultAndSitePages(t *testing.T) {
	routes := []models.Route{
		{RouteID: 1, Name: "Login", Path: "/auth/login", Type: models.RouteTypeController, Controller: "auth", ControllerAction: "login", Status: models.StatusActive},
		{RouteID: 2, Name: "Home", Path: "/", Type: models.RouteTypeController, Controller: "content", ControllerAction: "featured", Status: models.StatusActive, IsDefault: true},
		{RouteID: 3, Name: "Blog", Path: "/blog", Type: models.RouteTypeController, Controller: "content", ControllerAction: "category", Status: models.StatusActive},
		{RouteID: 4, Name: "Contact Us", Path: "/contact", Type: models.RouteTypeExtension, Status: models.StatusActive},
		{RouteID: 5, Name: "About", Path: "/info/about", Type: models.RouteTypeController, Controller: "content", ControllerAction: "item", Status: models.StatusActive},
		{RouteID: 6, Name: "Content Item", Path: "/content/item/*", Type: models.RouteTypeController, Controller: "content", ControllerAction: "item", Status: models.StatusActive},
		{RouteID: 7, Name: "Register", Path: "/register", Type: models.RouteTypeURL, Status: models.StatusActive},
	}
	sections := buildBlockRouteSections(routes)
	byKey := map[string]blockRouteSection{}
	for _, section := range sections {
		byKey[section.Key] = section
	}
	if section, ok := byKey["default"]; !ok || len(section.Routes) != 1 || section.Routes[0].RouteID != 2 {
		t.Fatalf("default section = %#v", section)
	}
	if section, ok := byKey["content-items"]; !ok || len(section.Routes) != 1 || section.Routes[0].RouteID != 5 {
		t.Fatalf("content-items section = %#v", byKey["content-items"])
	}
	if section, ok := byKey["site"]; !ok || len(section.Routes) != 1 || section.Routes[0].RouteID != 3 {
		t.Fatalf("site section = %#v", byKey["site"])
	}
	if section, ok := byKey["content"]; !ok || len(section.Routes) != 1 || section.Routes[0].RouteID != 6 {
		t.Fatalf("built-in content section = %#v", byKey["content"])
	}
	if section, ok := byKey["auth"]; !ok || len(section.Routes) != 1 || section.Routes[0].RouteID != 1 {
		t.Fatalf("auth section = %#v", byKey["auth"])
	}
	if section, ok := byKey["extensions"]; !ok || len(section.Routes) != 1 || section.Routes[0].RouteID != 4 {
		t.Fatalf("extensions section = %#v", byKey["extensions"])
	}
	if section, ok := byKey["urls"]; !ok || len(section.Routes) != 1 || section.Routes[0].RouteID != 7 {
		t.Fatalf("urls section = %#v", byKey["urls"])
	}
}
