package router

import (
	"strings"
	"testing"

	"github.com/rob121/cannon/internal/models"
)

func TestIframeRouteTypeConstant(t *testing.T) {
	if models.RouteTypeIframe != "Iframe" {
		t.Fatalf("RouteTypeIframe = %q", models.RouteTypeIframe)
	}
}

func TestIframeRouteRequiresTarget(t *testing.T) {
	route := models.Route{Type: models.RouteTypeIframe, Target: "  "}
	if strings.TrimSpace(route.Target) != "" {
		t.Fatal("expected empty trimmed target")
	}
}
