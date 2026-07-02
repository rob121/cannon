package blocks

import (
	"context"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestWrapDebugSpace(t *testing.T) {
	got := WrapDebugSpace("header", "<p>Hi</p>")
	for _, part := range []string{`border:1px dashed #FF3300`, ">header<", "<p>Hi</p>"} {
		if !strings.Contains(got, part) {
			t.Fatalf("WrapDebugSpace missing %q: %q", part, got)
		}
	}
}

func TestDebugSpacesActiveRequiresQueryAndSetting(t *testing.T) {
	ctx := context.Background()
	r := httptest.NewRequest("GET", "/?tp=1", nil)
	if DebugSpacesActive(ctx, r) {
		t.Fatal("expected false without global setting enabled")
	}
}
