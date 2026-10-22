package blocks

import (
	"context"
	"html/template"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/rob121/cannon/internal/cache"
	"github.com/rob121/cannon/internal/content"
	"github.com/rob121/cannon/internal/extensions"
	"github.com/rob121/cannon/internal/groups"
	"github.com/rob121/cannon/internal/hooks"
	"github.com/rob121/cannon/internal/markdown"
	"github.com/rob121/cannon/internal/models"
	"github.com/rob121/cannon/internal/sites"
	"gorm.io/gorm"
)

// FragmentRenderer executes a frontend template fragment without layout.
type FragmentRenderer func(name string, data map[string]any) (string, error)

// blockRenderParallel is how many blocks in one template space may render at once.
// Matches the SQLite reader pool so content queries and extension calls overlap
// instead of waiting on each other. GET/HEAD only; other methods stay in order
// because they share a request body.
const blockRenderParallel = 4

// RenderSpace renders all active blocks assigned to a template space.
func RenderSpace(ctx context.Context, extMgr *extensions.Manager, space string, r *http.Request, userCtx map[string]any, render FragmentRenderer) (string, error) {
	space = strings.TrimSpace(space)
	if space == "" {
		return "", nil
	}

	db, err := sites.DB(ctx)
	if err != nil {
		return "", err
	}

	rows, err := ListForSpace(ctx, db, space)
	if err != nil {
		return "", err
	}
	if len(rows) == 0 {
		return "", nil
	}

	parts := renderRows(ctx, extMgr, rows, space, r, userCtx, render)
	return strings.Join(parts, "\n"), nil
}

func renderRows(ctx context.Context, extMgr *extensions.Manager, rows []models.Block, space string, r *http.Request, userCtx map[string]any, render FragmentRenderer) []string {
	if len(rows) < 2 || !readOnlyRequest(r) {
		return renderRowsSequential(ctx, extMgr, rows, space, r, userCtx, render)
	}

	parts := make([]string, len(rows))
	var wg sync.WaitGroup
	sem := make(chan struct{}, blockRenderParallel)
	for i, row := range rows {
		wg.Add(1)
		sem <- struct{}{}
		go func(i int, row models.Block) {
			defer wg.Done()
			defer func() { <-sem }()
			parts[i] = renderRowHTML(ctx, extMgr, row, space, r, userCtx, render)
		}(i, row)
	}
	wg.Wait()

	out := make([]string, 0, len(parts))
	for _, html := range parts {
		if strings.TrimSpace(html) != "" {
			out = append(out, html)
		}
	}
	return out
}

func renderRowsSequential(ctx context.Context, extMgr *extensions.Manager, rows []models.Block, space string, r *http.Request, userCtx map[string]any, render FragmentRenderer) []string {
	parts := make([]string, 0, len(rows))
	for _, row := range rows {
		if html := renderRowHTML(ctx, extMgr, row, space, r, userCtx, render); strings.TrimSpace(html) != "" {
			parts = append(parts, html)
		}
	}
	return parts
}

func renderRowHTML(ctx context.Context, extMgr *extensions.Manager, row models.Block, space string, r *http.Request, userCtx map[string]any, render FragmentRenderer) string {
	html, err := renderOne(ctx, extMgr, row, space, r, userCtx, render)
	if err != nil {
		// Skip a broken block instead of taking down the whole page.
		log.Printf("blocks: space %q block_id=%d type=%s extension=%q: %v",
			space, row.BlockID, row.Type, row.ExtensionName, err)
		return ""
	}
	return html
}

func readOnlyRequest(r *http.Request) bool {
	if r == nil {
		return false
	}
	switch r.Method {
	case http.MethodGet, http.MethodHead:
		return true
	default:
		return false
	}
}

// ListForSpace returns active blocks for a template space ordered by sort then id.
func ListForSpace(ctx context.Context, db *gorm.DB, space string) ([]models.Block, error) {
	viewerGroups, err := groups.ViewerGroupIDs(ctx)
	if err != nil {
		return nil, err
	}

	rows, err := cache.BlocksForSpace(ctx, db, space)
	if err != nil {
		return nil, err
	}

	filtered := make([]models.Block, 0, len(rows))
	routeID := currentRouteID(ctx)
	now := time.Now()
	for _, row := range rows {
		if !groups.CanView(viewerGroups, row.Groups) {
			continue
		}
		meta, err := ParseMetadata(row.Metadata)
		if err != nil {
			return nil, err
		}
		if !PublishVisible(meta, now) {
			continue
		}
		if !RouteVisible(meta, routeID) {
			continue
		}
		filtered = append(filtered, row)
	}
	return filtered, nil
}

func renderOne(ctx context.Context, extMgr *extensions.Manager, row models.Block, space string, r *http.Request, userCtx map[string]any, render FragmentRenderer) (string, error) {
	meta, err := ParseMetadata(row.Metadata)
	if err != nil {
		return "", err
	}

	blockArgs := map[string]any{
		"block_id":   row.BlockID,
		"block_type": row.Type,
		"space":      space,
		"extension":  row.ExtensionName,
		"block_item": row.ExtensionBlockID,
	}
	if _, err := hooks.Fire(ctx, hooks.OnRenderBlock, blockArgs); err != nil {
		return "", err
	}

	var html string
	switch row.Type {
	case models.BlockTypeHTML:
		prepareArgs := map[string]any{
			"content":      meta.Content,
			"content_type": "block_html",
			"block_id":     row.BlockID,
			"space":        space,
		}
		if out, err := hooks.Fire(ctx, hooks.OnContentPrepare, prepareArgs); err != nil {
			return "", err
		} else if v, ok := out["content"].(string); ok {
			html = v
		} else {
			html = meta.Content
		}
	case models.BlockTypeMarkdown:
		if strings.TrimSpace(meta.Content) == "" {
			return "", nil
		}
		prepareArgs := map[string]any{
			"content":      meta.Content,
			"content_type": "block_markdown",
			"block_id":     row.BlockID,
			"space":        space,
		}
		if out, err := hooks.Fire(ctx, hooks.OnContentPrepare, prepareArgs); err != nil {
			return "", err
		} else if v, ok := out["content"].(string); ok {
			meta.Content = v
		}
		html, err = markdown.ToHTML(meta.Content)
		if err != nil {
			return "", err
		}
	case models.BlockTypeExtension:
		blockData, err := MetadataMap(row.Metadata)
		if err != nil {
			return "", err
		}
		item := strings.TrimSpace(row.ExtensionBlockID)
		if item == "" {
			return "", nil
		}
		html, err = extMgr.InvokeBlock(ctx, row.ExtensionName, item, space, r, userCtx, blockData)
		if err != nil {
			return "", err
		}
	case models.BlockTypeContent:
		viewerGroups, err := groups.ViewerGroupIDs(ctx)
		if err != nil {
			return "", err
		}
		html, err = content.RenderContentBlock(ctx, viewerGroups, content.ContentBlockOptions{
			Mode:         meta.ContentMode,
			Limit:        meta.Limit,
			CategorySlug: meta.CategorySlug,
			TagSlug:      meta.TagSlug,
			AuthorKey:    meta.AuthorKey,
			ItemSlug:     meta.ItemSlug,
			Layout:       meta.Layout,
		})
		if err != nil {
			return "", err
		}
	case models.BlockTypeLogin:
		html, err = RenderLoginBlock(ctx, r, BlockRow{BlockID: row.BlockID, Name: row.Name}, meta, render)
		if err != nil {
			return "", err
		}
	case models.BlockTypeMenuVertical:
		html, err = RenderMenuVerticalBlock(ctx, meta, render)
		if err != nil {
			return "", err
		}
	case models.BlockTypeMenuHorizontal:
		html, err = RenderMenuHorizontalBlock(ctx, meta, render)
		if err != nil {
			return "", err
		}
	case models.BlockTypeSearchHorizontal:
		html, err = RenderSearchHorizontalBlock(ctx, BlockRow{BlockID: row.BlockID}, meta, render)
		if err != nil {
			return "", err
		}
	case models.BlockTypeSearchVertical:
		html, err = RenderSearchVerticalBlock(ctx, BlockRow{BlockID: row.BlockID}, meta, render)
		if err != nil {
			return "", err
		}
	case models.BlockTypeAdminLiveAnalytics,
		models.BlockTypeAdminReviewQueue,
		models.BlockTypeAdminUpdateCheck,
		models.BlockTypeAdminLatestSignups,
		models.BlockTypeAdminQuickStart,
		models.BlockTypeAdminSystemReload:
		html, err = renderAdminDashboardBlock(ctx, row, r, render)
		if err != nil {
			return "", err
		}
		// Widgets ship their own card chrome; skip the default frontend wrapper.
		if strings.TrimSpace(meta.TemplateWrapper) == "" {
			meta.TemplateWrapper = PlainWrapperTemplate
			meta.ShowName = false
		}
	default:
		return "", nil
	}

	afterArgs := map[string]any{
		"block_id": row.BlockID,
		"space":    space,
		"html":     html,
	}
	if out, err := hooks.Fire(ctx, hooks.OnAfterRenderBlock, afterArgs); err != nil {
		return "", err
	} else if v, ok := out["html"].(string); ok {
		html = v
	}
	return finishBlockHTML(row, meta, html, space, render)
}

func finishBlockHTML(row models.Block, meta Metadata, body, space string, render FragmentRenderer) (string, error) {
	body = strings.TrimSpace(body)
	if body == "" && !meta.ShowName {
		return "", nil
	}
	wrapper := ResolveBlockTemplate(meta.TemplateWrapper)
	if wrapper == "" {
		wrapper = CardWrapperTemplate
	}
	if render == nil {
		return body, nil
	}
	wrapped, err := render(wrapper, blockWrapperData(row, meta, body, space))
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(wrapped), nil
}

func blockWrapperData(row models.Block, meta Metadata, body, space string) map[string]any {
	if strings.TrimSpace(space) == "" {
		space = PrimarySpace(row)
	}
	return map[string]any{
		"Name":     row.Name,
		"Body":     template.HTML(body),
		"Content":  template.HTML(body),
		"ShowName": meta.ShowName,
		"Space":    space,
		"Type":     string(row.Type),
		"BlockID":  row.BlockID,
	}
}
