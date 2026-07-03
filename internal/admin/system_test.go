package admin

import "testing"

func TestSystemPathParts(t *testing.T) {
	parts := pathParts("/system", "/system/reload")
	if len(parts) != 1 || parts[0] != "reload" {
		t.Fatalf("pathParts(/system/reload) = %v, want [reload]", parts)
	}
	parts = pathParts("/system", "/system/access-log/tail")
	if len(parts) != 2 || parts[0] != "access-log" || parts[1] != "tail" {
		t.Fatalf("pathParts(/system/access-log/tail) = %v", parts)
	}
	parts = pathParts("/system", "/system")
	if parts != nil {
		t.Fatalf("pathParts(/system) = %v, want nil", parts)
	}
}
