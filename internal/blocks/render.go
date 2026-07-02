package blocks

import (
	"context"
	"net/http"
	"strings"

	"github.com/rob121/cannon/internal/content"
	"github.com/rob121/cannon/internal/extensions"
	"github.com/rob121/cannon/internal/groups"
	"github.com/rob121/cannon/internal/hooks"
	"github.com/rob121/cannon/internal/markdown"
	"github.com/rob121/cannon/internal/models"
	"github.com/rob121/cannon/internal/sites"
	"gorm.io/gorm"
)

// RenderSpace renders all active blocks assigned to a template space.
func RenderSpace(ctx context.Context, extMgr *extensions.Manager, space string, r *http.Request, userCtx map[string]any) (string, error) {
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

	var parts []string
	for _, row := range rows {
		html, err := renderOne(ctx, extMgr, row, space, r, userCtx)
		if err != nil {
			return "", err
		}
		if strings.TrimSpace(html) != "" {
			parts = append(parts, html)
		}
	}
	return strings.Join(parts, "\n"), nil
}

// ListForSpace returns active blocks for a template space ordered by sort then id.
func ListForSpace(ctx context.Context, db *gorm.DB, space string) ([]models.Block, error) {
	viewerGroups, err := groups.ViewerGroupIDs(ctx)
	if err != nil {
		return nil, err
	}

	var rows []models.Block
	err = db.Preload("Groups").Where("space = ? AND status = ?", space, models.StatusActive).
		Order("sort asc, block_id asc").
		Find(&rows).Error
	if err != nil {
		return nil, err
	}

	filtered := make([]models.Block, 0, len(rows))
	for _, row := range rows {
		if groups.CanView(viewerGroups, row.Groups) {
			filtered = append(filtered, row)
		}
	}
	return filtered, nil
}

// DistinctSpaces returns unique space names used by blocks.
func DistinctSpaces(db *gorm.DB) ([]string, error) {
	var spaces []string
	err := db.Model(&models.Block{}).Distinct().Order("space asc").Pluck("space", &spaces).Error
	return spaces, err
}

func renderOne(ctx context.Context, extMgr *extensions.Manager, row models.Block, space string, r *http.Request, userCtx map[string]any) (string, error) {
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
	return html, nil
}
