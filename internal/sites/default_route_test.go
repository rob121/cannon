package sites

import (
	"testing"

	"github.com/rob121/cannon/internal/models"
)

func TestConcreteRoutePathFixedPath(t *testing.T) {
	got := ConcreteRoutePath(models.Route{Path: "/", ControllerAction: "category"})
	if got != "/" {
		t.Fatalf("got %q, want /", got)
	}
	got = ConcreteRoutePath(models.Route{Path: "/news", ControllerAction: "category"})
	if got != "/news" {
		t.Fatalf("got %q, want /news", got)
	}
}

func TestConcreteRoutePathWildcardUsesCategorySlug(t *testing.T) {
	got := ConcreteRoutePath(models.Route{
		Path:             "/content/category/*",
		Controller:       "content",
		ControllerAction: "category",
		Metadata:         `{"category_slug":"home"}`,
	})
	if got != "/content/category/home" {
		t.Fatalf("got %q, want /content/category/home", got)
	}
}

func TestConcreteRoutePathWildcardWithoutSlugUsesPrefix(t *testing.T) {
	got := ConcreteRoutePath(models.Route{
		Path:             "/content/category/*",
		ControllerAction: "category",
		Metadata:         `{}`,
	})
	if got != "/content/category" {
		t.Fatalf("got %q, want /content/category", got)
	}
}

func TestConcreteRoutePathFeedWithCategory(t *testing.T) {
	got := ConcreteRoutePath(models.Route{
		Path:             "/content/feed/*",
		ControllerAction: "feed",
		Metadata:         `{"feed_kind":"category","category_slug":"news"}`,
	})
	if got != "/content/feed/category/news" {
		t.Fatalf("got %q, want /content/feed/category/news", got)
	}
}
