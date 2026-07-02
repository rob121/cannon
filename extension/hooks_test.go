package extension

import (
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHookAbort(t *testing.T) {
	resp := HookAbort("denied")
	if !resp.Stop {
		t.Fatal("expected stop")
	}
	args := HookArguments(HookWireRequest{Arguments: resp.Arguments})
	if args["allowed"] != false || args["error"] != "denied" {
		t.Fatalf("args=%v", args)
	}
}

func TestDecodeHookWireRequest(t *testing.T) {
	body := `{"event":"onBeforeRoute","arguments":{"path":"/"},"site_id":"demo","method":"GET","url":"/"}`
	req := httptest.NewRequest("POST", "/hooks", strings.NewReader(body))
	req.ContentLength = int64(len(body))
	got, err := DecodeHookWireRequest(req, "fallback")
	if err != nil {
		t.Fatal(err)
	}
	if got.Event != "onBeforeRoute" || got.SiteID != "demo" {
		t.Fatalf("req=%+v", got)
	}
}
