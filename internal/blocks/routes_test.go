package blocks

import "testing"

func TestRouteVisible(t *testing.T) {
	tests := []struct {
		name     string
		meta     Metadata
		routeID  uint
		expected bool
	}{
		{name: "all default", meta: Metadata{}, routeID: 5, expected: true},
		{name: "none", meta: Metadata{RouteMode: RouteModeNone}, routeID: 5, expected: false},
		{name: "only match", meta: Metadata{RouteMode: RouteModeOnly, RouteIDs: []uint{5, 7}}, routeID: 5, expected: true},
		{name: "only miss", meta: Metadata{RouteMode: RouteModeOnly, RouteIDs: []uint{5, 7}}, routeID: 3, expected: false},
		{name: "except match", meta: Metadata{RouteMode: RouteModeExcept, RouteIDs: []uint{5}}, routeID: 5, expected: false},
		{name: "except miss", meta: Metadata{RouteMode: RouteModeExcept, RouteIDs: []uint{5}}, routeID: 7, expected: true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := RouteVisible(tc.meta, tc.routeID); got != tc.expected {
				t.Fatalf("RouteVisible() = %v, want %v", got, tc.expected)
			}
		})
	}
}

func TestMetadataFromFormValuesRouteAssignment(t *testing.T) {
	values := map[string][]string{
		"route_mode": {"only"},
		"route_ids":  {"3", "7", "7"},
	}
	raw, err := MetadataFromFormValues("html", "hello", values)
	if err != nil {
		t.Fatal(err)
	}
	meta, err := ParseMetadata(raw)
	if err != nil {
		t.Fatal(err)
	}
	if meta.RouteMode != RouteModeOnly {
		t.Fatalf("route_mode = %q", meta.RouteMode)
	}
	if len(meta.RouteIDs) != 2 || meta.RouteIDs[0] != 3 || meta.RouteIDs[1] != 7 {
		t.Fatalf("route_ids = %v", meta.RouteIDs)
	}
}
