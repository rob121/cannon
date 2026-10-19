package extension

import (
	"encoding/json"
	"net/http"
	"testing"
)

func TestWithEmitAppendsEvent(t *testing.T) {
	resp := WithEmit(Redirect(http.StatusSeeOther, "/thanks"), "onSubmissionAfterCreate", map[string]any{
		"title": "Hello",
	})
	if resp.StatusCode != http.StatusSeeOther {
		t.Fatalf("status = %d", resp.StatusCode)
	}
	if len(resp.EmitEvents) != 1 || resp.EmitEvents[0].Event != "onSubmissionAfterCreate" {
		t.Fatalf("emit = %#v", resp.EmitEvents)
	}
	raw, err := json.Marshal(resp)
	if err != nil {
		t.Fatal(err)
	}
	var decoded WireResponse
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatal(err)
	}
	if len(decoded.EmitEvents) != 1 || decoded.EmitEvents[0].Arguments["title"] != "Hello" {
		t.Fatalf("decoded = %#v", decoded.EmitEvents)
	}
}

func TestWithEmitIgnoresEmptyEvent(t *testing.T) {
	resp := WithEmit(OK(), "  ", nil)
	if len(resp.EmitEvents) != 0 {
		t.Fatalf("emit = %#v", resp.EmitEvents)
	}
}
