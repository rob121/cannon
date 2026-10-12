package content

import (
	"context"
	"errors"

	"github.com/rob121/cannon/internal/models"
	"github.com/rob121/cannon/internal/settings"
	"github.com/rob121/cannon/internal/sites"
	"gorm.io/gorm"
)

const (
	DefaultCategoryListColumns      = 3
	DefaultCategoryListPageSize     = 20
	FeaturedCategorySettingsSection = "featured_category"
)

// CategoryListingSettings controls how items render on a category page.
type CategoryListingSettings struct {
	Columns      int
	Pagination   bool
	PageSize     int
	ShowReadMore bool
	LeadingItems int
}

// FeaturedCategoryDefaults returns the virtual category used by the featured item listing.
func FeaturedCategoryDefaults(ctx context.Context) models.Category {
	return models.Category{
		Name:               "Featured",
		Slug:               "featured",
		Description:        "",
		Template:           "",
		InheritSettings:    true,
		InheritPermissions: true,
		ShowTitle:          true,
		ShowDescription:    true,
		ShowReadMore:       false,
		ItemTitleDisplay:   DisplayInherit,
		ItemAuthorDisplay:  DisplayInherit,
		ItemDateDisplay:    DisplayInherit,
		LeadingArticles:    0,
		ListColumns:        DefaultCategoryListColumns,
		ListPagination:     true,
		ListPageSize:       DefaultCategoryListPageSize,
		Status:             models.StatusActive,
		Locale:             LocaleFromContext(ctx),
	}
}

// FeaturedCategorySettings loads the virtual category settings used by /content/featured.
func FeaturedCategorySettings(ctx context.Context) (models.Category, error) {
	row := FeaturedCategoryDefaults(ctx)
	data, err := settings.NewStore().Load(ctx, settings.ScopeGlobal, FeaturedCategorySettingsSection)
	if err != nil {
		return row, err
	}
	if v := settingsString(data, "name", ""); v != "" {
		row.Name = v
	}
	row.Description = settingsString(data, "description", row.Description)
	row.Image = settingsString(data, "image", row.Image)
	row.Template = settingsString(data, "template", row.Template)
	row.InheritSettings = settingsBool(data, "inherit_settings", row.InheritSettings)
	row.ShowTitle = settingsBool(data, "show_title", row.ShowTitle)
	row.ShowDescription = settingsBool(data, "show_description", row.ShowDescription)
	row.ShowReadMore = settingsBool(data, "show_read_more", row.ShowReadMore)
	row.ItemTitleDisplay = settingsDisplayMode(data, "item_title_display", row.ItemTitleDisplay)
	row.ItemAuthorDisplay = settingsDisplayMode(data, "item_author_display", row.ItemAuthorDisplay)
	row.ItemDateDisplay = settingsDisplayMode(data, "item_date_display", row.ItemDateDisplay)
	row.LeadingArticles = NormalizeCategoryLeadingArticles(settingsInt(data, "leading_articles", row.LeadingArticles))
	row.ListColumns = NormalizeCategoryListColumns(settingsInt(data, "list_columns", row.ListColumns))
	row.ListPagination = settingsBool(data, "list_pagination", row.ListPagination)
	row.ListPageSize = NormalizeCategoryListPageSize(settingsInt(data, "list_page_size", row.ListPageSize))
	return row, nil
}

// SaveFeaturedCategorySettings stores the virtual category settings used by /content/featured.
func SaveFeaturedCategorySettings(ctx context.Context, row models.Category) error {
	if row.Name == "" {
		row.Name = "Featured"
	}
	data := map[string]any{
		"name":                row.Name,
		"description":         row.Description,
		"image":               row.Image,
		"template":            row.Template,
		"inherit_settings":    row.InheritSettings,
		"show_title":          row.ShowTitle,
		"show_description":    row.ShowDescription,
		"show_read_more":      row.ShowReadMore,
		"item_title_display":  NormalizeDisplayMode(row.ItemTitleDisplay),
		"item_author_display": NormalizeDisplayMode(row.ItemAuthorDisplay),
		"item_date_display":   NormalizeDisplayMode(row.ItemDateDisplay),
		"leading_articles":    NormalizeCategoryLeadingArticles(row.LeadingArticles),
		"list_columns":        NormalizeCategoryListColumns(row.ListColumns),
		"list_pagination":     row.ListPagination,
		"list_page_size":      NormalizeCategoryListPageSize(row.ListPageSize),
	}
	return settings.NewStore().Save(ctx, settings.ScopeGlobal, FeaturedCategorySettingsSection, data)
}

// FeaturedShowTitle resolves whether the featured listing should display its title.
func FeaturedShowTitle(ctx context.Context, cat models.Category) bool {
	return cat.ShowTitle
}

// FeaturedShowDescription resolves whether the featured listing should display its description.
func FeaturedShowDescription(ctx context.Context, cat models.Category) bool {
	return cat.ShowDescription
}

// ResolveFeaturedListingSettings resolves columns, pagination, and page size for the featured listing.
func ResolveFeaturedListingSettings(ctx context.Context, cat models.Category) CategoryListingSettings {
	return CategoryListingSettings{
		Columns:      NormalizeCategoryListColumns(cat.ListColumns),
		Pagination:   cat.ListPagination,
		PageSize:     NormalizeCategoryListPageSize(cat.ListPageSize),
		ShowReadMore: cat.ShowReadMore,
		LeadingItems: NormalizeCategoryLeadingArticles(cat.LeadingArticles),
	}
}

// ResolveFeaturedItemDisplaySettings resolves item summary display settings for /content/featured.
func ResolveFeaturedItemDisplaySettings(ctx context.Context, cat models.Category, base Settings) ItemDisplaySettings {
	return ItemDisplaySettings{
		ShowTitle:         ResolveDisplayMode(cat.ItemTitleDisplay, base.ShowTitle),
		ShowAuthor:        ResolveDisplayMode(cat.ItemAuthorDisplay, base.ShowAuthor),
		ShowPublishedDate: ResolveDisplayMode(cat.ItemDateDisplay, base.ShowPublishedDate),
	}
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
			Columns:      DefaultCategoryListColumns,
			Pagination:   true,
			PageSize:     DefaultCategoryListPageSize,
			ShowReadMore: false,
			LeadingItems: 0,
		}, nil
	}
	return CategoryListingSettings{
		Columns:      NormalizeCategoryListColumns(resolved.ListColumns),
		Pagination:   resolved.ListPagination,
		PageSize:     NormalizeCategoryListPageSize(resolved.ListPageSize),
		ShowReadMore: resolved.ShowReadMore,
		LeadingItems: NormalizeCategoryLeadingArticles(resolved.LeadingArticles),
	}, nil
}

// ResolveCategoryItemDisplaySettings resolves item display flags for items rendered in a category context.
func ResolveCategoryItemDisplaySettings(ctx context.Context, cat *models.Category, base Settings) (ItemDisplaySettings, error) {
	resolved, err := ResolveCategorySettings(ctx, cat)
	if err != nil {
		return ItemDisplaySettings{}, err
	}
	if resolved == nil {
		return ItemDisplayFromSettings(base), nil
	}
	return ItemDisplaySettings{
		ShowTitle:         ResolveDisplayMode(resolved.ItemTitleDisplay, base.ShowTitle),
		ShowAuthor:        ResolveDisplayMode(resolved.ItemAuthorDisplay, base.ShowAuthor),
		ShowPublishedDate: ResolveDisplayMode(resolved.ItemDateDisplay, base.ShowPublishedDate),
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

// NormalizeCategoryLeadingArticles clamps the number of full-width leading articles to 0–25.
func NormalizeCategoryLeadingArticles(count int) int {
	if count < 0 {
		return 0
	}
	if count > 25 {
		return 25
	}
	return count
}

func settingsString(data map[string]any, key, fallback string) string {
	v, ok := data[key]
	if !ok || v == nil {
		return fallback
	}
	if s, ok := v.(string); ok {
		return s
	}
	return fallback
}

func settingsInt(data map[string]any, key string, fallback int) int {
	v, ok := data[key]
	if !ok || v == nil {
		return fallback
	}
	switch n := v.(type) {
	case float64:
		return int(n)
	case int:
		return n
	case int64:
		return int(n)
	}
	return fallback
}

// SplitCategoryLeadingItems separates the current page's items into full-width leading items and the remaining column items.
func SplitCategoryLeadingItems(items []models.Item, settings CategoryListingSettings, page int) ([]models.Item, []models.Item) {
	leading := NormalizeCategoryLeadingArticles(settings.LeadingItems)
	if leading <= 0 || len(items) == 0 {
		return nil, items
	}
	if page < 1 {
		page = 1
	}
	pageSize := NormalizeCategoryListPageSize(settings.PageSize)
	offset := 0
	if settings.Pagination {
		offset = (page - 1) * pageSize
	}
	remainingLeading := leading - offset
	if remainingLeading <= 0 {
		return nil, items
	}
	if remainingLeading > len(items) {
		remainingLeading = len(items)
	}
	return items[:remainingLeading], items[remainingLeading:]
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

// CategoryItemListClass returns CSS classes for semantic item listing layouts.
func CategoryItemListClass(columns int) string {
	switch NormalizeCategoryListColumns(columns) {
	case 1:
		return "site-item-list"
	case 2:
		return "site-item-list site-item-list--cols-2"
	case 4:
		return "site-item-list site-item-list--cols-4"
	default:
		return "site-item-list site-item-list--grid"
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
