package content

import (
	"strings"
	"testing"
	"time"

	"github.com/rob121/cannon/internal/models"
)

func TestBuildRSSUsesAuthorDisplayName(t *testing.T) {
	raw, err := BuildRSS("Test Site", "https://example.test", "", "", "", []models.Item{
		{
			Title:     "Post",
			Slug:      "post",
			CreatedAt: time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC),
			Author: &models.User{
				GivenName:  "Ada",
				FamilyName: "Lovelace",
				Username:   "ada",
			},
		},
	})
	if err != nil {
		t.Fatalf("BuildRSS() error = %v", err)
	}
	xml := string(raw)
	if !strings.Contains(xml, "<author>Ada Lovelace</author>") {
		t.Fatalf("expected author display name in RSS, got:\n%s", xml)
	}
	if strings.Contains(xml, "<author>ada</author>") {
		t.Fatalf("RSS leaked username as author:\n%s", xml)
	}
}
