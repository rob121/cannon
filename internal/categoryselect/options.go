package categoryselect

import (
	"strings"

	"github.com/rob121/cannon/internal/models"
)

// Option is a category entry for hierarchical select controls.
type Option struct {
	models.Category
	Label string
	Depth int
}

// Flatten returns categories in tree order with indented labels.
func Flatten(rows []models.Category) []Option {
	if len(rows) == 0 {
		return nil
	}
	roots, byParent := treeBuckets(rows)
	out := make([]Option, 0, len(rows))
	seen := make(map[uint]struct{}, len(rows))
	for _, root := range roots {
		out = appendOptions(out, root, byParent, 0, seen)
	}
	for _, row := range rows {
		if _, ok := seen[row.CategoryID]; ok {
			continue
		}
		out = append(out, Option{
			Category: row,
			Label:    indentedLabel(row.Name, 0),
			Depth:    0,
		})
	}
	return out
}

// ParentOptions returns tree-ordered categories valid as a parent choice.
// The excludeID category and its descendants are omitted.
func ParentOptions(rows []models.Category, excludeID uint) []Option {
	flat := Flatten(rows)
	if excludeID == 0 {
		return flat
	}
	excluded := descendantSet(rows, excludeID)
	out := make([]Option, 0, len(flat))
	for _, opt := range flat {
		if excluded[opt.CategoryID] {
			continue
		}
		out = append(out, opt)
	}
	return out
}

func indentedLabel(name string, depth int) string {
	if depth <= 0 {
		return name
	}
	return strings.Repeat("— ", depth) + name
}

func treeBuckets(rows []models.Category) ([]models.Category, map[uint][]models.Category) {
	byParent := make(map[uint][]models.Category)
	roots := make([]models.Category, 0)
	for _, row := range rows {
		if row.ParentID == nil || *row.ParentID == 0 {
			roots = append(roots, row)
			continue
		}
		pid := *row.ParentID
		byParent[pid] = append(byParent[pid], row)
	}
	return roots, byParent
}

func appendOptions(out []Option, cat models.Category, byParent map[uint][]models.Category, depth int, seen map[uint]struct{}) []Option {
	seen[cat.CategoryID] = struct{}{}
	out = append(out, Option{
		Category: cat,
		Label:    indentedLabel(cat.Name, depth),
		Depth:    depth,
	})
	for _, child := range byParent[cat.CategoryID] {
		out = appendOptions(out, child, byParent, depth+1, seen)
	}
	return out
}

func descendantSet(rows []models.Category, rootID uint) map[uint]bool {
	if rootID == 0 {
		return map[uint]bool{}
	}
	children := make(map[uint][]uint)
	for _, row := range rows {
		if row.ParentID != nil && *row.ParentID > 0 {
			children[*row.ParentID] = append(children[*row.ParentID], row.CategoryID)
		}
	}
	out := map[uint]bool{rootID: true}
	var walk func(id uint)
	walk = func(id uint) {
		for _, childID := range children[id] {
			if out[childID] {
				continue
			}
			out[childID] = true
			walk(childID)
		}
	}
	walk(rootID)
	return out
}
