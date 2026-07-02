package extension

import "testing"

func TestCSRFHelpers(t *testing.T) {
	req := WireRequest{CSRF: `tok"en`}
	if CSRFToken(req) != `tok"en` {
		t.Fatal("CSRFToken mismatch")
	}
	got := CSRFHiddenField(req)
	want := `<input type="hidden" name="_csrf" value="tok&#34;en">`
	if got != want {
		t.Fatalf("CSRFHiddenField = %q, want %q", got, want)
	}
	if CSRFHiddenField(WireRequest{}) != "" {
		t.Fatal("empty token should render no field")
	}
}
