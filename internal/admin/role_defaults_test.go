package admin

import (
	"testing"

	"github.com/rob121/cannon/internal/models"
)

func TestApplyUnsetRoleDefaultsFillsBlankSystemRole(t *testing.T) {
	writer := models.Role{RoleID: 5, Name: "writer"}
	row := models.Role{RoleID: 2, Name: "editor"}
	row, parents, allows := applyUnsetRoleDefaults(row, []models.Role{writer, row}, nil, nil, nil)
	if row.Status != models.StatusActive {
		t.Fatalf("status = %q", row.Status)
	}
	if row.Description == "" {
		t.Fatal("expected built-in description")
	}
	if len(parents) != 1 || parents[0] != writer.RoleID {
		t.Fatalf("parents = %#v", parents)
	}
	if len(allows) == 0 {
		t.Fatal("expected built-in permissions")
	}
}

func TestApplyUnsetRoleDefaultsKeepsSavedValues(t *testing.T) {
	row := models.Role{
		RoleID:      2,
		Name:        "editor",
		Description: "Custom description",
		Status:      models.StatusInactive,
	}
	row, parents, allows := applyUnsetRoleDefaults(row, nil, []uint{9}, []string{"core.admin.access"}, nil)
	if row.Description != "Custom description" || row.Status != models.StatusInactive {
		t.Fatalf("row = %#v", row)
	}
	if len(parents) != 1 || parents[0] != 9 {
		t.Fatalf("parents = %#v", parents)
	}
	if len(allows) != 1 || allows[0] != "core.admin.access" {
		t.Fatalf("allows = %#v", allows)
	}
}

func TestApplyUnsetRoleDefaultsLeavesCustomRoleBlank(t *testing.T) {
	row := models.Role{Name: "camp host"}
	row, parents, allows := applyUnsetRoleDefaults(row, nil, nil, nil, nil)
	if row.Status != models.StatusActive {
		t.Fatalf("status = %q", row.Status)
	}
	if row.Description != "" || len(parents) != 0 || len(allows) != 0 {
		t.Fatalf("row=%#v parents=%#v allows=%#v", row, parents, allows)
	}
}
