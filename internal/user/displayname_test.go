package user

import (
	"testing"

	"github.com/rob121/cannon/internal/models"
)

func TestDisplayName(t *testing.T) {
	if got := DisplayName(nil); got != "" {
		t.Fatalf("nil user: got %q", got)
	}
	if got := DisplayName(&models.User{GivenName: "Ada", FamilyName: "Lovelace", Username: "ada"}); got != "Ada Lovelace" {
		t.Fatalf("full name: got %q", got)
	}
	if got := DisplayName(&models.User{Username: "ada"}); got != "ada" {
		t.Fatalf("username fallback: got %q", got)
	}
}
