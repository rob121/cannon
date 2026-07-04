package content

import (
	"context"
	"fmt"
	"html"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/rob121/cannon/internal/models"
	"github.com/rob121/cannon/internal/sites"
)

// ContentBlockOptions configures a content block render.
type ContentBlockOptions struct {
	Mode         string
	Limit        int
	CategorySlug string
	TagSlug      string
	AuthorKey    string
	ItemSlug     string
	Layout       string
}

// RenderContentBlock returns HTML for a CMS content listing block.
func RenderContentBlock(ctx context.Context, viewerGroups []uint, opts ContentBlockOptions) (string, error) {
	opts.Mode = strings.TrimSpace(strings.ToLower(opts.Mode))
	if opts.Mode == "" {
		opts.Mode = "latest"
	}
	if opts.Limit <= 0 {
		opts.Limit = 5
	}
	layout := strings.TrimSpace(strings.ToLower(opts.Layout))
	if layout == "" {
		layout = "list"
	}

	var (
		items []models.Item
		err   error
	)
	switch opts.Mode {
	case "latest":
		items, _, err = ListItems(ctx, viewerGroups, ListOptions{Page: 1, Limit: opts.Limit})
	case "featured":
		items, _, err = ListItems(ctx, viewerGroups, ListOptions{Featured: true, Page: 1, Limit: opts.Limit})
	case "popular":
		items, err = PopularItems(ctx, viewerGroups, opts.Limit)
	case "category":
		items, err = itemsForCategorySlug(ctx, viewerGroups, opts.CategorySlug, opts.Limit)
	case "tag":
		items, err = itemsForTagSlug(ctx, viewerGroups, opts.TagSlug, opts.Limit)
	case "author":
		items, err = itemsForAuthorKey(ctx, viewerGroups, opts.AuthorKey, opts.Limit)
	case "related":
		items, err = relatedForSlug(ctx, viewerGroups, opts.ItemSlug, opts.Limit)
	case "archive":
		return renderArchiveBlock(ctx, viewerGroups, opts.Limit)
	case "tag_cloud":
		return renderTagCloud(ctx, viewerGroups, opts.Limit)
	case "category_menu":
		return renderCategoryMenu(ctx)
	default:
		return "", fmt.Errorf("unknown content block mode %q", opts.Mode)
	}
	if err != nil {
		return "", err
	}
	return renderItemBlock(items, layout), nil
}

func itemsForCategorySlug(ctx context.Context, viewerGroups []uint, slug string, limit int) ([]models.Item, error) {
	cat, err := CategoryBySlug(ctx, slug, viewerGroups)
	if err != nil {
		return nil, err
	}
	items, _, err := ListItems(ctx, viewerGroups, ListOptions{CategoryID: cat.CategoryID, Page: 1, Limit: limit})
	return items, err
}

func itemsForTagSlug(ctx context.Context, viewerGroups []uint, slug string, limit int) ([]models.Item, error) {
	tag, err := TagBySlug(ctx, slug)
	if err != nil {
		return nil, err
	}
	items, _, err := ListItems(ctx, viewerGroups, ListOptions{TagID: tag.TagID, Page: 1, Limit: limit})
	return items, err
}

func itemsForAuthorKey(ctx context.Context, viewerGroups []uint, key string, limit int) ([]models.Item, error) {
	db, err := sites.DB(ctx)
	if err != nil {
		return nil, err
	}
	var user models.User
	if id, convErr := strconv.ParseUint(key, 10, 64); convErr == nil {
		err = db.Where("user_id = ? AND status = ?", id, models.StatusActive).First(&user).Error
	} else {
		err = db.Where("username = ? AND status = ?", key, models.StatusActive).First(&user).Error
	}
	if err != nil {
		return nil, err
	}
	items, _, err := ListItems(ctx, viewerGroups, ListOptions{AuthorID: user.UserID, Page: 1, Limit: limit})
	return items, err
}

func relatedForSlug(ctx context.Context, viewerGroups []uint, slug string, limit int) ([]models.Item, error) {
	item, err := ItemBySlug(ctx, slug, viewerGroups)
	if err != nil {
		return nil, err
	}
	return RelatedItems(ctx, viewerGroups, item, limit)
}

func renderItemBlock(items []models.Item, layout string) string {
	if len(items) == 0 {
		return `<p class="text-muted mb-0">No items to display.</p>`
	}
	var b strings.Builder
	if layout == "grid" {
		b.WriteString(`<div class="row g-3">`)
		for _, item := range items {
			b.WriteString(`<div class="col-12 col-md-6"><article class="card h-100 shadow-sm"><div class="card-body position-relative">`)
			b.WriteString(`<h3 class="card-title h6 mb-2"><a href="`)
			b.WriteString(html.EscapeString(ItemURLForContext(WithLocale(context.Background(), item.Locale), item.Slug)))
			b.WriteString(`" class="stretched-link text-decoration-none">`)
			b.WriteString(html.EscapeString(item.Title))
			b.WriteString(`</a></h3>`)
			if intro := strings.TrimSpace(item.Intro); intro != "" {
				b.WriteString(`<p class="card-text small text-muted mb-0">`)
				b.WriteString(html.EscapeString(intro))
				b.WriteString(`</p>`)
			}
			b.WriteString(`</div></article></div>`)
		}
		b.WriteString(`</div>`)
		return b.String()
	}
	b.WriteString(`<div class="list-group list-group-flush">`)
	for _, item := range items {
		b.WriteString(`<a class="list-group-item list-group-item-action px-0" href="`)
		b.WriteString(html.EscapeString(ItemURLForContext(WithLocale(context.Background(), item.Locale), item.Slug)))
		b.WriteString(`"><h3 class="h6 mb-1">`)
		b.WriteString(html.EscapeString(item.Title))
		b.WriteString(`</h3>`)
		if intro := strings.TrimSpace(item.Intro); intro != "" {
			b.WriteString(`<p class="mb-0 small text-muted">`)
			b.WriteString(html.EscapeString(intro))
			b.WriteString(`</p>`)
		}
		b.WriteString(`</a>`)
	}
	b.WriteString(`</div>`)
	return b.String()
}

func renderTagCloud(ctx context.Context, viewerGroups []uint, limit int) (string, error) {
	cloud, err := TagCloud(ctx, viewerGroups, limit)
	if err != nil {
		return "", err
	}
	if len(cloud) == 0 {
		return `<p class="text-muted mb-0">No tags yet.</p>`, nil
	}
	var b strings.Builder
	b.WriteString(`<div class="d-flex flex-wrap gap-2">`)
	for _, row := range cloud {
		b.WriteString(`<a class="badge text-bg-light border text-decoration-none" href="`)
		b.WriteString(html.EscapeString(TagURL(row.Tag.Slug)))
		b.WriteString(`">`)
		b.WriteString(html.EscapeString(row.Tag.Name))
		b.WriteString(` <span class="text-muted">(`)
		b.WriteString(strconv.FormatInt(row.Count, 10))
		b.WriteString(`)</span></a>`)
	}
	b.WriteString(`</div>`)
	return b.String(), nil
}

func renderCategoryMenu(ctx context.Context) (string, error) {
	categories, err := CategoryTree(ctx)
	if err != nil {
		return "", err
	}
	if len(categories) == 0 {
		return `<p class="text-muted mb-0">No categories yet.</p>`, nil
	}
	byParent := map[uint][]models.Category{}
	roots := make([]models.Category, 0)
	for _, cat := range categories {
		if cat.ParentID == nil || *cat.ParentID == 0 {
			roots = append(roots, cat)
			continue
		}
		byParent[*cat.ParentID] = append(byParent[*cat.ParentID], cat)
	}
	var b strings.Builder
	b.WriteString(`<ul class="list-unstyled mb-0">`)
	writeCategoryNodes(&b, roots, byParent)
	b.WriteString(`</ul>`)
	return b.String(), nil
}

func writeCategoryNodes(b *strings.Builder, nodes []models.Category, byParent map[uint][]models.Category) {
	for _, cat := range nodes {
		b.WriteString(`<li><a href="`)
		b.WriteString(html.EscapeString(CategoryURLForContext(WithLocale(context.Background(), cat.Locale), cat.Slug)))
		b.WriteString(`">`)
		b.WriteString(html.EscapeString(cat.Name))
		b.WriteString(`</a>`)
		if children := byParent[cat.CategoryID]; len(children) > 0 {
			b.WriteString(`<ul class="list-unstyled ps-3 mb-0">`)
			writeCategoryNodes(b, children, byParent)
			b.WriteString(`</ul>`)
		}
		b.WriteString(`</li>`)
	}
}

type archiveMonth struct {
	Key   string
	Label string
	Count int
}

func renderArchiveBlock(ctx context.Context, viewerGroups []uint, limit int) (string, error) {
	if limit <= 0 {
		limit = 12
	}
	items, _, err := ListItems(ctx, viewerGroups, ListOptions{Page: 1, Limit: 2000})
	if err != nil {
		return "", err
	}
	counts := map[string]int{}
	for _, item := range items {
		when := item.CreatedAt
		if item.PublishStart != nil && !item.PublishStart.IsZero() {
			when = *item.PublishStart
		}
		if when.IsZero() {
			continue
		}
		key := when.Format("2006-01")
		counts[key]++
	}
	months := make([]archiveMonth, 0, len(counts))
	for key, count := range counts {
		t, parseErr := time.Parse("2006-01", key)
		label := key
		if parseErr == nil {
			label = t.Format("January 2006")
		}
		months = append(months, archiveMonth{Key: key, Label: label, Count: count})
	}
	sort.Slice(months, func(i, j int) bool { return months[i].Key > months[j].Key })
	if len(months) > limit {
		months = months[:limit]
	}
	if len(months) == 0 {
		return `<p class="text-muted mb-0">No archived content yet.</p>`, nil
	}
	var b strings.Builder
	b.WriteString(`<ul class="list-unstyled mb-0">`)
	for _, row := range months {
		b.WriteString(`<li class="d-flex justify-content-between gap-2 py-1"><span>`)
		b.WriteString(html.EscapeString(row.Label))
		b.WriteString(`</span><span class="text-muted">(`)
		b.WriteString(strconv.Itoa(row.Count))
		b.WriteString(`)</span></li>`)
	}
	b.WriteString(`</ul>`)
	return b.String(), nil
}
