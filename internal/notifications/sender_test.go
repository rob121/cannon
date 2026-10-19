package notifications

import (
	"context"
	"strings"
	"testing"
)

func TestSendShoutrrRejectsUnknownScheme(t *testing.T) {
	err := sendShoutrr(context.Background(), "notaservice://token@host", "", "hello")
	if err == nil {
		t.Fatal("expected error for unknown scheme")
	}
}

func TestSendShoutrrAcceptsPushoverURLShape(t *testing.T) {
	// Locate/CreateSender should accept a well-formed pushover URL even if send fails.
	err := sendShoutrr(context.Background(), "pushover://shoutrrr:apitoken@userkey/?devices=phone", "Title", "body")
	if err == nil {
		t.Fatal("expected send failure without real credentials")
	}
	msg := err.Error()
	if strings.Contains(msg, `unsupported protocol scheme "pushover"`) {
		t.Fatalf("fell through to generic HTTP: %v", err)
	}
	if strings.Contains(msg, "invalid shoutrrr url") {
		t.Fatalf("pushover URL should be recognized: %v", err)
	}
}
