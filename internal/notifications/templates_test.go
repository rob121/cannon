package notifications

import (
	"testing"

	"github.com/rob121/cannon/internal/models"
)

func TestRenderEventMessageDefaultAndTemplate(t *testing.T) {
	args := map[string]any{"username": "rob", "email": "rob@example.com"}
	subject, body := RenderEventMessage("", "onUserSignup", args)
	if subject != "" {
		t.Fatalf("default subject = %q", subject)
	}
	if body != "Cannon: onUserSignup username=rob email=rob@example.com" {
		t.Fatalf("default body = %q", body)
	}

	templates := TemplatesToJSON(map[string]EventTemplate{
		"onUserSignup": {Subject: "New user {{username}}", Body: "Email: {{email}}"},
	})
	subject, body = RenderEventMessage(templates, "onUserSignup", args)
	if subject != "New user rob" || body != "Email: rob@example.com" {
		t.Fatalf("templated = %q / %q", subject, body)
	}
}

func TestFlattenArgsItem(t *testing.T) {
	author := uint(9)
	cat := uint(3)
	item := models.Item{
		ItemID:     12,
		Title:      "Hello",
		Slug:       "hello",
		Status:     models.ItemStatusPending,
		AuthorID:   &author,
		CategoryID: &cat,
		Category:   &models.Category{CategoryID: 3, Slug: "news"},
	}
	vars := FlattenArgs(map[string]any{"item": item, "item_id": item.ItemID})
	if vars["status"] != "pending" || vars["author_id"] != "9" || vars["category"] != "news" {
		t.Fatalf("vars = %#v", vars)
	}
}

func TestMatchFilters(t *testing.T) {
	author := uint(5)
	item := models.Item{Status: models.ItemStatusPending, AuthorID: &author, CategoryID: uintPtr(2)}
	args := map[string]any{"item": item}

	if !MatchFilters(nil, "", args, 5) {
		t.Fatal("empty filters should match")
	}
	if MatchFilters(nil, FiltersToJSON(SubscriptionFilters{Status: "published"}), args, 5) {
		t.Fatal("status mismatch should fail")
	}
	if !MatchFilters(nil, FiltersToJSON(SubscriptionFilters{Status: "pending", AuthorSelf: true}), args, 5) {
		t.Fatal("pending + author_self should match")
	}
	if MatchFilters(nil, FiltersToJSON(SubscriptionFilters{AuthorSelf: true}), args, 99) {
		t.Fatal("wrong author should fail")
	}
	if !MatchFilters(nil, FiltersToJSON(SubscriptionFilters{Category: "2"}), args, 5) {
		t.Fatal("category id should match")
	}
}

func uintPtr(v uint) *uint { return &v }
