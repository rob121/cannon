package hooks

import "testing"

func TestIsEmittable(t *testing.T) {
	// Core EmittableEvents is intentionally empty; extension events use the
	// notifications capability catalog instead.
	if IsEmittable(OnUserBeforeLogin) {
		t.Fatal("login hooks must not be emittable")
	}
	if IsEmittable("") {
		t.Fatal("empty event must not be emittable")
	}
	for _, event := range EmittableEvents {
		if !IsEmittable(event) {
			t.Fatalf("expected %q to be emittable", event)
		}
	}
}
