package accesslog

import "testing"

func TestHostKey(t *testing.T) {
	if got := HostKey("http://localhost:8001"); got != "localhost-8001" {
		t.Fatalf("HostKey() = %q", got)
	}
	if got := HostKey("127.0.0.1:8001"); got != "127.0.0.1-8001" {
		t.Fatalf("HostKey() = %q", got)
	}
}
