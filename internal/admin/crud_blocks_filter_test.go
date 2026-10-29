package admin

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/rob121/cannon/internal/models"
)

func TestBlockMatchesListFilters(t *testing.T) {
	row := models.Block{
		Name: "Footer Links",
		Type: models.BlockTypeHTML,
	}

	if !blockMatchesListFilters(row, "", "") {
		t.Fatal("expected empty filters to match")
	}
	if !blockMatchesListFilters(row, string(models.BlockTypeHTML), "") {
		t.Fatal("expected type filter match")
	}
	if blockMatchesListFilters(row, string(models.BlockTypeMarkdown), "") {
		t.Fatal("expected type filter mismatch")
	}
	if !blockMatchesListFilters(row, "", "footer") {
		t.Fatal("expected name search match")
	}
	if !blockMatchesListFilters(row, "", "html") {
		t.Fatal("expected type search match")
	}
	if blockMatchesListFilters(row, "", "missing") {
		t.Fatal("expected search miss")
	}
}

func TestBlockListFiltersFromRequestRejectsUnknownType(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/admin/blocks?block_type=unknown&q=+sidebar+", nil)
	blockType, query := blockListFiltersFromRequest(r)
	if blockType != "" {
		t.Fatalf("block_type = %q, want empty", blockType)
	}
	if query != "sidebar" {
		t.Fatalf("query = %q, want sidebar", query)
	}
}
