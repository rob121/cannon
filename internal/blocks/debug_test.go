package blocks

import (
	"context"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestWrapDebugSpace(t *testing.T) {
	got := WrapDebugSpace("header", "<p>Hi</p>")
	for _, part := range []string{`border:2px dashed #FF3300`, "min-height:2rem", ">header<", "<p>Hi</p>"} {
		if !strings.Contains(got, part) {
			t.Fatalf("WrapDebugSpace missing %q: %q", part, got)
		}
	}
}

func TestWrapDebugEmptySpace(t *testing.T) {
	got := WrapDebugSpace("empty", "")
	for _, part := range []string{`border:2px dashed #FF3300`, ">empty</div></div>"} {
		if !strings.Contains(got, part) {
			t.Fatalf("empty space missing %q: %q", part, got)
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
