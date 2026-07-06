package admin

import (
	"context"
	"testing"
)

func TestNavCanMapReturnsNonNilMap(t *testing.T) {
	can := navCanMap(context.Background())
	if can == nil {
		t.Fatal("expected non-nil nav permission map")
	}
}
