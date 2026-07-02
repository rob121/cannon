package content

import (
	"context"
	"fmt"
	"html"
	"strconv"
	"strings"

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
		return `<p class="site-block-empty">No items to display.</p>`
	}
	containerClass := "site-block-items site-block-items--list"
	itemClass := "site-block-item"
	if layout == "grid" {
		containerClass = "site-block-items site-block-items--grid"
		itemClass = "site-block-item site-block-item--card"
	}
	var b strings.Builder
	b.WriteString(`<div class="`)
	b.WriteString(containerClass)
	b.WriteString(`">`)
	for _, item := range items {
		b.WriteString(`<article class="`)
		b.WriteString(itemClass)
		b.WriteString(`"><h3><a href="`)
		b.WriteString(html.EscapeString(ItemURL(item.Slug)))
		b.WriteString(`">`)
		b.WriteString(html.EscapeString(item.Title))
		b.WriteString(`</a></h3>`)
		if intro := strings.TrimSpace(item.Intro); intro != "" {
			b.WriteString(`<p>`)
			b.WriteString(html.EscapeString(intro))
			b.WriteString(`</p>`)
		}
		b.WriteString(`</article>`)
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
		return `<p class="site-block-empty">No tags yet.</p>`, nil
	}
	var b strings.Builder
	b.WriteString(`<ul class="site-tag-cloud">`)
	for _, row := range cloud {
		b.WriteString(`<li class="site-tag-cloud-item"><a href="`)
		b.WriteString(html.EscapeString(TagURL(row.Tag.Slug)))
		b.WriteString(`">`)
		b.WriteString(html.EscapeString(row.Tag.Name))
		b.WriteString(`</a> <span class="site-tag-count">(`)
		b.WriteString(strconv.FormatInt(row.Count, 10))
		b.WriteString(`)</span></li>`)
	}
	b.WriteString(`</ul>`)
	return b.String(), nil
}

func renderCategoryMenu(ctx context.Context) (string, error) {
	categories, err := CategoryTree(ctx)
	if err != nil {
		return "", err
	}
	if len(categories) == 0 {
		return `<p class="site-block-empty">No categories yet.</p>`, nil
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
	b.WriteString(`<ul class="site-category-menu">`)
	writeCategoryNodes(&b, roots, byParent)
	b.WriteString(`</ul>`)
	return b.String(), nil
}

func writeCategoryNodes(b *strings.Builder, nodes []models.Category, byParent map[uint][]models.Category) {
	for _, cat := range nodes {
		b.WriteString(`<li><a href="`)
		b.WriteString(html.EscapeString(CategoryURL(cat.Slug)))
		b.WriteString(`">`)
		b.WriteString(html.EscapeString(cat.Name))
		b.WriteString(`</a>`)
		if children := byParent[cat.CategoryID]; len(children) > 0 {
			b.WriteString(`<ul>`)
			writeCategoryNodes(b, children, byParent)
			b.WriteString(`</ul>`)
		}
		b.WriteString(`</li>`)
	}
}
