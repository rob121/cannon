package content

import (
	"net/http"
	"testing"

	"github.com/rob121/cannon/internal/controllers"
	"github.com/rob121/cannon/internal/models"
)

func TestRouteContentSlugPrefersMetaOnFixedHomePath(t *testing.T) {
	req, err := http.NewRequest(http.MethodGet, "/", nil)
	if err != nil {
		t.Fatal(err)
	}
	ctx := &controllers.Context{
		Request: req,
		Route: models.Route{
			Path:             "/",
			Controller:       ControllerID,
			ControllerAction: ActionCategory,
			Metadata:         `{"category_slug":"home"}`,
		},
	}
	if got := routeContentSlug(ctx, "category_slug"); got != "home" {
		t.Fatalf("got %q, want home", got)
	}
}

func TestRouteContentSlugPrefersPathSuffixOnWildcard(t *testing.T) {
	req, err := http.NewRequest(http.MethodGet, "/content/category/from-path", nil)
	if err != nil {
		t.Fatal(err)
	}
	ctx := &controllers.Context{
		Request: req,
		Route: models.Route{
			Path:             "/content/category/*",
			Controller:       ControllerID,
			ControllerAction: ActionCategory,
			Metadata:         `{"category_slug":"from-meta"}`,
		},
	}
	if got := routeContentSlug(ctx, "category_slug"); got != "from-path" {
		t.Fatalf("got %q, want from-path", got)
	}
}

func TestRouteContentSlugFallsBackToMetaWhenWildcardSuffixEmpty(t *testing.T) {
	req, err := http.NewRequest(http.MethodGet, "/content/category", nil)
	if err != nil {
		t.Fatal(err)
	}
	ctx := &controllers.Context{
		Request: req,
		Route: models.Route{
			Path:             "/content/category/*",
			Controller:       ControllerID,
			ControllerAction: ActionCategory,
			Metadata:         `{"category_slug":"predefined"}`,
		},
	}
	if got := routeContentSlug(ctx, "category_slug"); got != "predefined" {
		t.Fatalf("got %q, want predefined", got)
	}
}

func TestCategoryPaginationBaseURLUsesFixedRoutePath(t *testing.T) {
	req, err := http.NewRequest(http.MethodGet, "/", nil)
	if err != nil {
		t.Fatal(err)
	}
	ctx := &controllers.Context{
		Request: req,
		Route: models.Route{
			Path:             "/",
			ControllerAction: ActionCategory,
			Metadata:         `{"category_slug":"home"}`,
		},
	}
	if got := categoryPaginationBaseURL(ctx, "home"); got != "/" {
		t.Fatalf("got %q, want /", got)
	}
}
