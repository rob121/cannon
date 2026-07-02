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
	}
	sections := buildBlockRouteSections(routes)
	if len(sections) < 3 {
		t.Fatalf("sections = %d, want at least 3", len(sections))
	}
	if sections[0].Key != "default" || len(sections[0].Routes) != 1 || sections[0].Routes[0].RouteID != 2 {
		t.Fatalf("default section = %#v", sections[0])
	}

	var foundBlog bool
	for _, section := range sections {
		if section.Key != "site" {
			continue
		}
		for _, route := range section.Routes {
			if route.RouteID == 3 {
				foundBlog = true
			}
		}
	}
	if !foundBlog {
		t.Fatal("expected custom site page /blog in site section")
	}
}
