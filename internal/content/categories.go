package content

import (
	"context"

	"github.com/rob121/cannon/internal/models"
)

// CategoryDescendantIDs returns a category ID and all active descendant category IDs.
func CategoryDescendantIDs(ctx context.Context, rootID uint) ([]uint, error) {
	if rootID == 0 {
		return nil, nil
	}
	rows, err := CategoryTree(ctx)
	if err != nil {
		return nil, err
	}
	children := make(map[uint][]uint)
	for _, row := range rows {
		if row.ParentID == nil || *row.ParentID == 0 {
			continue
		}
		children[*row.ParentID] = append(children[*row.ParentID], row.CategoryID)
	}
	out := []uint{rootID}
	var walk func(id uint)
	walk = func(id uint) {
		for _, childID := range children[id] {
			out = append(out, childID)
			walk(childID)
		}
	}
	walk(rootID)
	return out, nil
}

// FieldGroupForCategory resolves the effective field group for a category,
// using a direct assignment or inherited parent settings.
func FieldGroupForCategory(ctx context.Context, cat *models.Category) (*uint, error) {
	if cat == nil {
		return nil, nil
	}
	if cat.FieldGroupID != nil && *cat.FieldGroupID > 0 {
		return cat.FieldGroupID, nil
	}
	if !cat.InheritSettings || cat.ParentID == nil || *cat.ParentID == 0 {
		return nil, nil
	}
	resolved, err := ResolveCategorySettings(ctx, cat)
	if err != nil {
		return nil, err
	}
	if resolved == nil {
		return nil, nil
	}
	return resolved.FieldGroupID, nil
}
