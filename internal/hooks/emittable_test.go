package hooks

import "testing"

func TestIsEmittable(t *testing.T) {
	if !IsEmittable(OnSubmissionAfterCreate) {
		t.Fatal("expected onSubmissionAfterCreate to be emittable")
	}
	if IsEmittable(OnUserBeforeLogin) {
		t.Fatal("login hooks must not be emittable")
	}
	if IsEmittable("") {
		t.Fatal("empty event must not be emittable")
	}
}
