package admin

import (
	"testing"

	"github.com/rob121/cannon/internal/models"
)

func TestPaginateRoutes(t *testing.T) {
	rows := []models.Route{
		{RouteID: 1},
		{RouteID: 2},
		{RouteID: 3},
	}

	page := paginateRoutes(rows, 2, 2)
	if len(page) != 1 || page[0].RouteID != 3 {
		t.Fatalf("page 2 = %+v", page)
	}

	page = paginateRoutes(rows, 3, 2)
	if len(page) != 0 {
		t.Fatalf("out of range page = %+v", page)
	}
}
