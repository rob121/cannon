package content

import (
	"context"
	"errors"

	"github.com/rob121/cannon/internal/models"
	"github.com/rob121/cannon/internal/sites"
	"gorm.io/gorm"
)

const (
	DefaultCategoryListColumns  = 3
	DefaultCategoryListPageSize = 20
)

// CategoryListingSettings controls how items render on a category page.
type CategoryListingSettings struct {
	Columns    int
	Pagination bool
	PageSize   int
}

// ResolveCategorySettings returns the category whose display settings apply,
// walking parents while InheritSettings is enabled.
func ResolveCategorySettings(ctx context.Context, cat *models.Category) (*models.Category, error) {
	if cat == nil {
		return nil, nil
	}
	db, err := sites.DB(ctx)
	if err != nil {
		return nil, err
	}
	current := cat
	for current.InheritSettings && current.ParentID != nil && *current.ParentID > 0 {
		var parent models.Category
		if err := db.First(&parent, *current.ParentID).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				break
			}
			return nil, err
		}
		current = &parent
	}
	return current, nil
}

// CategoryShowTitle resolves whether a category page should display its title.
func CategoryShowTitle(ctx context.Context, cat *models.Category) (bool, error) {
	resolved, err := ResolveCategorySettings(ctx, cat)
	if err != nil {
		return false, err
	}
	if resolved == nil {
		return true, nil
	}
	return resolved.ShowTitle, nil
}

// CategoryShowDescription resolves whether a category page should display its description.
func CategoryShowDescription(ctx context.Context, cat *models.Category) (bool, error) {
	resolved, err := ResolveCategorySettings(ctx, cat)
	if err != nil {
		return false, err
	}
	if resolved == nil {
		return true, nil
	}
	return resolved.ShowDescription, nil
}

// ResolveCategoryListingSettings resolves columns, pagination, and page size for a category listing.
func ResolveCategoryListingSettings(ctx context.Context, cat *models.Category) (CategoryListingSettings, error) {
	resolved, err := ResolveCategorySettings(ctx, cat)
	if err != nil {
		return CategoryListingSettings{}, err
	}
	if resolved == nil {
		return CategoryListingSettings{
			Columns:    DefaultCategoryListColumns,
			Pagination: true,
			PageSize:   DefaultCategoryListPageSize,
		}, nil
	}
	return CategoryListingSettings{
		Columns:    NormalizeCategoryListColumns(resolved.ListColumns),
		Pagination: resolved.ListPagination,
		PageSize:   NormalizeCategoryListPageSize(resolved.ListPageSize),
	}, nil
}

// NormalizeCategoryListColumns clamps category grid columns to 1–4.
func NormalizeCategoryListColumns(columns int) int {
	switch columns {
	case 1, 2, 3, 4:
		return columns
	default:
		return DefaultCategoryListColumns
	}
}

// NormalizeCategoryListPageSize clamps page size to supported listing sizes.
func NormalizeCategoryListPageSize(size int) int {
	switch size {
	case 6, 9, 12, 18, 20, 24, 30, 50:
		return size
	default:
		return DefaultCategoryListPageSize
	}
}

// CategoryItemColumnClass returns a Bootstrap column class for item cards.
func CategoryItemColumnClass(columns int) string {
	switch NormalizeCategoryListColumns(columns) {
	case 1:
		return "col-12"
	case 2:
		return "col-md-6"
	case 4:
		return "col-md-6 col-lg-3"
	default:
		return "col-md-6 col-lg-4"
	}
}

// ListTotalPages returns the page count for a paginated listing.
func ListTotalPages(total int64, pageSize int) int {
	if pageSize <= 0 {
		return 1
	}
	if total <= 0 {
		return 1
	}
	pages := int((total + int64(pageSize) - 1) / int64(pageSize))
	if pages < 1 {
		return 1
	}
	return pages
}
