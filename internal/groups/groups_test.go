package groups

import (
	"testing"

	"github.com/rob121/cannon/internal/models"
)

func TestCanView(t *testing.T) {
	public := models.Group{GroupID: 1, Name: PublicGroupName, Status: models.StatusActive}
	members := models.Group{GroupID: 2, Name: "members", Status: models.StatusActive}
	inactive := models.Group{GroupID: 3, Name: "staff", Status: models.StatusInactive}

	if !CanView([]uint{1}, []models.Group{public}) {
		t.Fatal("public viewer should see public content")
	}
	if CanView([]uint{1}, []models.Group{members}) {
		t.Fatal("public viewer should not see members-only content")
	}
	if !CanView([]uint{1, 2}, []models.Group{members}) {
		t.Fatal("member viewer should see members content")
	}
	if CanView([]uint{2}, []models.Group{inactive}) {
		t.Fatal("inactive content groups should not grant access")
	}
	if CanView([]uint{1}, nil) {
		t.Fatal("content without groups should be hidden")
	}
}

func TestCanViewContent(t *testing.T) {
	members := models.Group{GroupID: 2, Name: "members", Status: models.StatusActive}

	if !CanViewContent([]uint{1}, nil) {
		t.Fatal("unrestricted content should be visible")
	}
	if CanViewContent([]uint{1}, []models.Group{members}) {
		t.Fatal("public viewer should not see members-only content")
	}
	if !CanViewContent([]uint{1, 2}, []models.Group{members}) {
		t.Fatal("member viewer should see members content")
	}
}
