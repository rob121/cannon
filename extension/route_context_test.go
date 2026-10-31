package extension

import "testing"

func TestMountPathSuffix(t *testing.T) {
	tests := []struct {
		route, req, want string
	}{
		{"/jobs/*", "/jobs", ""},
		{"/jobs/*", "/jobs/coordinator", "coordinator"},
		{"/jobs", "/jobs", ""},
		{"/jobs", "/jobs/comm-coordinator", "comm-coordinator"},
	}
	for _, tc := range tests {
		if got := MountPathSuffix(tc.route, tc.req); got != tc.want {
			t.Fatalf("MountPathSuffix(%q, %q) = %q want %q", tc.route, tc.req, got, tc.want)
		}
	}
}
