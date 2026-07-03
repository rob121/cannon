package content

import (
	"context"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/rob121/cannon/internal/groups"
	"github.com/rob121/cannon/internal/models"
	"github.com/rob121/cannon/internal/sites"
	"gorm.io/gorm"
)

var slugRE = regexp.MustCompile(`[^a-z0-9]+`)

// Slugify converts a string to a URL-safe slug.
func Slugify(v string) string {
	v = strings.ToLower(strings.TrimSpace(v))
	v = slugRE.ReplaceAllString(v, "-")
	v = strings.Trim(v, "-")
	if v == "" {
		return "item"
	}
	return v
}

// UniqueItemSlug returns a slug unused by other items.
func UniqueItemSlug(ctx context.Context, base string, excludeID uint) (string, error) {
	db, err := sites.DB(ctx)
	if err != nil {
		return "", err
	}
	slug := Slugify(base)
	for i := 0; i < 100; i++ {
		candidate := slug
		if i > 0 {
			candidate = slug + "-" + strconv.Itoa(i)
		}
		var count int64
		q := db.Model(&models.Item{}).Where("slug = ?", candidate)
		if excludeID > 0 {
			q = q.Where("item_id <> ?", excludeID)
		}
		if err := q.Count(&count).Error; err != nil {
			return "", err
		}
		if count == 0 {
			return candidate, nil
		}
	}
	return slug + "-" + strconv.FormatInt(time.Now().Unix(), 10), nil
}

// UniqueCategorySlug returns a slug unused by other categories.
func UniqueCategorySlug(ctx context.Context, base string, excludeID uint) (string, error) {
	return ResolveCategorySlug(ctx, base, "", nil, excludeID)
}

// ResolveCategorySlug builds a unique category slug, prefixing with the parent path when set.
func ResolveCategorySlug(ctx context.Context, name, manualSlug string, parentID *uint, excludeID uint) (string, error) {
	base := strings.TrimSpace(manualSlug)
	if base == "" {
		var err error
		base, err = CategorySlugBase(ctx, name, parentID)
		if err != nil {
			return "", err
		}
	} else {
		base = NormalizeCategorySlug(base)
	}
	return uniquifyCategorySlug(ctx, base, excludeID)
}

// CategorySlugBase returns the preferred slug path for a category name and optional parent.
func CategorySlugBase(ctx context.Context, name string, parentID *uint) (string, error) {
	leaf := Slugify(name)
	if parentID == nil || *parentID == 0 {
		return leaf, nil
	}
	db, err := sites.DB(ctx)
	if err != nil {
		return "", err
	}
	var parent models.Category
	if err := db.Select("slug").First(&parent, *parentID).Error; err != nil {
		return leaf, nil
	}
	parentSlug := strings.Trim(strings.TrimSpace(parent.Slug), "/")
	if parentSlug == "" {
		return leaf, nil
	}
	return parentSlug + "/" + leaf, nil
}

// NormalizeCategorySlug cleans a user-provided category slug path.
func NormalizeCategorySlug(raw string) string {
	raw = strings.TrimSpace(raw)
	raw = strings.Trim(raw, "/")
	if raw == "" {
		return "category"
	}
	parts := strings.Split(raw, "/")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		part = Slugify(part)
		if part != "" {
			out = append(out, part)
		}
	}
	if len(out) == 0 {
		return "category"
	}
	return strings.Join(out, "/")
}

func uniquifyCategorySlug(ctx context.Context, base string, excludeID uint) (string, error) {
	db, err := sites.DB(ctx)
	if err != nil {
		return "", err
	}
	for i := 0; i < 100; i++ {
		candidate := categorySlugWithSuffix(base, i)
		var count int64
		q := db.Model(&models.Category{}).Where("slug = ?", candidate)
		if excludeID > 0 {
			q = q.Where("category_id <> ?", excludeID)
		}
		if err := q.Count(&count).Error; err != nil {
			return "", err
		}
		if count == 0 {
			return candidate, nil
		}
	}
	return categorySlugWithSuffix(base, int(time.Now().Unix()%1000)), nil
}

func categorySlugWithSuffix(path string, n int) string {
	if n == 0 {
		return path
	}
	suffix := "-" + strconv.Itoa(n)
	if idx := strings.LastIndex(path, "/"); idx >= 0 {
		return path[:idx+1] + path[idx+1:] + suffix
	}
	return path + suffix
}

// UniqueTagSlug returns a slug unused by other tags.
func UniqueTagSlug(ctx context.Context, base string, excludeID uint) (string, error) {
	db, err := sites.DB(ctx)
	if err != nil {
		return "", err
	}
	slug := Slugify(base)
	for i := 0; i < 100; i++ {
		candidate := slug
		if i > 0 {
			candidate = slug + "-" + strconv.Itoa(i)
		}
		var count int64
		q := db.Model(&models.Tag{}).Where("slug = ?", candidate)
		if excludeID > 0 {
			q = q.Where("tag_id <> ?", excludeID)
		}
		if err := q.Count(&count).Error; err != nil {
			return "", err
		}
		if count == 0 {
			return candidate, nil
		}
	}
	return slug + "-" + strconv.FormatInt(time.Now().Unix(), 10), nil
}

// PublishedScope limits queries to viewable published items.
func PublishedScope(db *gorm.DB, now time.Time) *gorm.DB {
	return db.Where("status = ?", models.ItemStatusPublished).
		Where("(publish_start IS NULL OR publish_start <= ?)", now).
		Where("(publish_end IS NULL OR publish_end >= ?)", now)
}

// VisibleItemsQuery returns published items visible to viewer groups.
func VisibleItemsQuery(ctx context.Context, viewerGroups []uint) (*gorm.DB, error) {
	db, err := sites.DB(ctx)
	if err != nil {
		return nil, err
	}
	now := time.Now()
	q := PublishedScope(db.Model(&models.Item{}), now).
		Preload("Category").
		Preload("Tags").
		Preload("Author").
		Preload("Groups")
	return filterByGroups(q, viewerGroups), nil
}

func filterByGroups(q *gorm.DB, viewerGroups []uint) *gorm.DB {
	if len(viewerGroups) == 0 {
		return q.Where(`NOT EXISTS (
			SELECT 1 FROM item_groups ig WHERE ig.item_item_id = items.item_id
		)`)
	}
	return q.Where(`NOT EXISTS (
			SELECT 1 FROM item_groups ig WHERE ig.item_item_id = items.item_id
		) OR EXISTS (
			SELECT 1 FROM item_groups ig
			WHERE ig.item_item_id = items.item_id AND ig.group_group_id IN ?
		)`, viewerGroups)
}

// ListItems returns paginated visible items with optional filters.
func ListItems(ctx context.Context, viewerGroups []uint, opts ListOptions) ([]models.Item, int64, error) {
	q, err := VisibleItemsQuery(ctx, viewerGroups)
	if err != nil {
		return nil, 0, err
	}
	if len(opts.CategoryIDs) > 0 {
		q = q.Where("category_id IN ?", opts.CategoryIDs)
	} else if opts.CategoryID > 0 {
		q = q.Where("category_id = ?", opts.CategoryID)
	}
	if opts.AuthorID > 0 {
		q = q.Where("author_id = ?", opts.AuthorID)
	}
	if opts.Featured {
		q = q.Where("featured = ?", true)
	}
	if opts.TagID > 0 {
		q = q.Joins("JOIN item_tags ON item_tags.item_item_id = items.item_id AND item_tags.tag_tag_id = ?", opts.TagID)
	}
	if strings.TrimSpace(opts.Query) != "" {
		like := "%" + strings.TrimSpace(opts.Query) + "%"
		q = q.Where("title LIKE ? OR intro LIKE ? OR body LIKE ?", like, like, like)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	order := "sort ASC, created_at DESC"
	if opts.Featured && (opts.Sort == "" || opts.Sort == "sort" || opts.Sort == "featured_sort") {
		if opts.Sort == "featured_sort" && opts.Dir == "desc" {
			order = "featured_sort DESC, created_at DESC"
		} else {
			order = "featured_sort ASC, created_at DESC"
		}
	} else {
		switch opts.Sort {
		case "title":
			if opts.Dir == "desc" {
				order = "title DESC"
			} else {
				order = "title ASC"
			}
		case "sort":
			if opts.Dir == "desc" {
				order = "sort DESC, created_at DESC"
			}
		case "featured":
			order = "featured DESC, featured_sort ASC, created_at DESC"
		case "featured_sort":
			if opts.Dir == "desc" {
				order = "featured_sort DESC, created_at DESC"
			} else {
				order = "featured_sort ASC, created_at DESC"
			}
		case "popular":
			order = "(SELECT COUNT(*) FROM comments WHERE comments.item_id = items.item_id AND comments.approved = 1) DESC, created_at DESC"
		}
	}
	page := opts.Page
	if page < 1 {
		page = 1
	}
	limit := opts.Limit
	if !opts.NoPagination && limit <= 0 {
		limit = 20
	}
	var rows []models.Item
	q = q.Order(order)
	if opts.NoPagination {
		if err := q.Find(&rows).Error; err != nil {
			return nil, 0, err
		}
	} else {
		if err := q.Offset((page - 1) * limit).Limit(limit).Find(&rows).Error; err != nil {
			return nil, 0, err
		}
	}
	return rows, total, nil
}

type ListOptions struct {
	CategoryID     uint
	CategoryIDs    []uint
	AuthorID       uint
	TagID          uint
	Query          string
	Featured       bool
	Sort           string
	Dir            string
	Page           int
	Limit          int
	NoPagination   bool
}

// ItemBySlug loads a published item by slug.
func ItemBySlug(ctx context.Context, slug string, viewerGroups []uint) (*models.Item, error) {
	q, err := VisibleItemsQuery(ctx, viewerGroups)
	if err != nil {
		return nil, err
	}
	var item models.Item
	if err := q.Where("slug = ?", slug).First(&item).Error; err != nil {
		return nil, err
	}
	if !groups.CanViewContent(viewerGroups, item.Groups) {
		return nil, gorm.ErrRecordNotFound
	}
	return &item, nil
}

// CategoryBySlug loads an active category by slug.
func CategoryBySlug(ctx context.Context, slug string, viewerGroups []uint) (*models.Category, error) {
	db, err := sites.DB(ctx)
	if err != nil {
		return nil, err
	}
	var cat models.Category
	if err := db.Preload("Groups").Where("slug = ? AND status = ?", slug, models.StatusActive).First(&cat).Error; err != nil {
		return nil, err
	}
	if !groups.CanViewContent(viewerGroups, cat.Groups) {
		return nil, gorm.ErrRecordNotFound
	}
	return &cat, nil
}

// TagBySlug loads a tag by slug.
func TagBySlug(ctx context.Context, slug string) (*models.Tag, error) {
	db, err := sites.DB(ctx)
	if err != nil {
		return nil, err
	}
	var tag models.Tag
	if err := db.Where("slug = ?", slug).First(&tag).Error; err != nil {
		return nil, err
	}
	return &tag, nil
}

// CategoryTree returns active categories sorted for display.
func CategoryTree(ctx context.Context) ([]models.Category, error) {
	db, err := sites.DB(ctx)
	if err != nil {
		return nil, err
	}
	var rows []models.Category
	if err := db.Where("status = ?", models.StatusActive).Order("sort ASC, name ASC").Find(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}

// FieldsForCategory returns active custom fields for a category's effective field group.
func FieldsForCategory(ctx context.Context, cat *models.Category) ([]models.ContentField, error) {
	groupID, err := FieldGroupForCategory(ctx, cat)
	if err != nil {
		return nil, err
	}
	if groupID == nil || *groupID == 0 {
		return nil, nil
	}
	db, err := sites.DB(ctx)
	if err != nil {
		return nil, err
	}
	var fields []models.ContentField
	if err := db.Where("field_group_id = ? AND status = ?", *groupID, models.StatusActive).
		Order("sort ASC, label ASC").Find(&fields).Error; err != nil {
		return nil, err
	}
	return fields, nil
}

// ItemFieldMap returns custom field values keyed by field name.
func ItemFieldMap(ctx context.Context, itemID uint) (map[string]string, error) {
	db, err := sites.DB(ctx)
	if err != nil {
		return nil, err
	}
	var values []models.ItemFieldValue
	if err := db.Where("item_id = ?", itemID).Find(&values).Error; err != nil {
		return nil, err
	}
	if len(values) == 0 {
		return map[string]string{}, nil
	}
	ids := make([]uint, 0, len(values))
	for _, v := range values {
		ids = append(ids, v.FieldID)
	}
	var fields []models.ContentField
	if err := db.Where("field_id IN ?", ids).Find(&fields).Error; err != nil {
		return nil, err
	}
	byID := map[uint]string{}
	for _, f := range fields {
		byID[f.FieldID] = f.Name
	}
	out := make(map[string]string, len(values))
	for _, v := range values {
		if name, ok := byID[v.FieldID]; ok {
			out[name] = v.Value
		}
	}
	return out, nil
}

// ItemFieldDisplays returns labeled custom fields with formatted HTML for item pages.
func ItemFieldDisplays(ctx context.Context, item *models.Item) ([]FieldDisplay, error) {
	if item == nil || item.ItemID == 0 {
		return nil, nil
	}
	var cat *models.Category
	if item.CategoryID != nil && *item.CategoryID > 0 {
		cat = item.Category
	}
	fields, err := FieldsForCategory(ctx, cat)
	if err != nil || len(fields) == 0 {
		return nil, err
	}
	values, err := ItemFieldMap(ctx, item.ItemID)
	if err != nil {
		return nil, err
	}
	out := make([]FieldDisplay, 0, len(fields))
	for _, field := range fields {
		value := values[field.Name]
		if strings.TrimSpace(value) == "" {
			continue
		}
		label := strings.TrimSpace(field.Label)
		if label == "" {
			label = field.Name
		}
		out = append(out, FieldDisplay{
			Name:  field.Name,
			Label: label,
			Type:  field.Type,
			Value: value,
			HTML:  FormatFieldDisplayHTML(field, value),
		})
	}
	return out, nil
}

// ApprovedComments returns approved comments for an item.
func ApprovedComments(ctx context.Context, itemID uint) ([]models.Comment, error) {
	db, err := sites.DB(ctx)
	if err != nil {
		return nil, err
	}
	var rows []models.Comment
	if err := db.Where("item_id = ? AND approved = ?", itemID, true).
		Order("created_at ASC").Find(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}
