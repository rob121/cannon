package admin

import "testing"

func TestSystemPathParts(t *testing.T) {
	parts := pathParts("/system", "/system/reload")
	if len(parts) != 1 || parts[0] != "reload" {
		t.Fatalf("pathParts(/system/reload) = %v, want [reload]", parts)
	}
	parts = pathParts("/system", "/system")
	if parts != nil {
		t.Fatalf("pathParts(/system) = %v, want nil", parts)
	}
}
