package content_test

import (
	"testing"

	"github.com/rob121/cannon/internal/content"
)

func TestSlugify(t *testing.T) {
	if got := content.Slugify("Hello World!"); got != "hello-world" {
		t.Fatalf("got %q", got)
	}
}
