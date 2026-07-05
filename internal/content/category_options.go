package content

import (
	"github.com/rob121/cannon/internal/categoryselect"
	"github.com/rob121/cannon/internal/models"
)

// CategoryOption is a category entry for hierarchical select controls.
type CategoryOption = categoryselect.Option

// FlattenCategoryOptions returns categories in tree order with indented labels.
func FlattenCategoryOptions(rows []models.Category) []CategoryOption {
	return categoryselect.Flatten(rows)
}

// CategoryParentOptions returns tree-ordered categories valid as a parent choice.
func CategoryParentOptions(rows []models.Category, excludeID uint) []CategoryOption {
	return categoryselect.ParentOptions(rows, excludeID)
}
