package admin

import (
	"net/http"
	"sort"
	"strings"

	"github.com/rob121/cannon/internal/blocks"
	"github.com/rob121/cannon/internal/models"
)

func blockListFiltersFromRequest(r *http.Request) (blockType, query string) {
	q := r.URL.Query()
	blockType = strings.TrimSpace(q.Get("block_type"))
	if !isKnownBlockType(blockType) {
		blockType = ""
	}
	query = strings.TrimSpace(q.Get("q"))
	return blockType, query
}

func allBlockTypes() []models.BlockType {
	return []models.BlockType{
		models.BlockTypeHTML,
		models.BlockTypeMarkdown,
		models.BlockTypeExtension,
		models.BlockTypeContent,
		models.BlockTypeLogin,
		models.BlockTypeMenuVertical,
		models.BlockTypeMenuHorizontal,
		models.BlockTypeSearchHorizontal,
		models.BlockTypeSearchVertical,
		models.BlockTypeAdminLiveAnalytics,
		models.BlockTypeAdminReviewQueue,
		models.BlockTypeAdminUpdateCheck,
		models.BlockTypeAdminLatestSignups,
		models.BlockTypeAdminQuickStart,
		models.BlockTypeAdminSystemReload,
	}
}

func isKnownBlockType(t string) bool {
	if t == "" {
		return false
	}
	for _, bt := range allBlockTypes() {
		if string(bt) == t {
			return true
		}
	}
	return false
}

func blockTypeOptionLabel(t models.BlockType) string {
	switch t {
	case models.BlockTypeHTML:
		return "HTML"
	case models.BlockTypeMarkdown:
		return "Markdown"
	case models.BlockTypeExtension:
		return "Extension"
	case models.BlockTypeContent:
		return "Content"
	case models.BlockTypeLogin:
		return "Login"
	case models.BlockTypeMenuVertical:
		return "Menu Vertical"
	case models.BlockTypeMenuHorizontal:
		return "Menu Horizontal"
	case models.BlockTypeSearchHorizontal:
		return "Search Horizontal"
	case models.BlockTypeSearchVertical:
		return "Search Vertical"
	case models.BlockTypeAdminLiveAnalytics:
		return "Admin · Live Analytics"
	case models.BlockTypeAdminReviewQueue:
		return "Admin · Review Queue"
	case models.BlockTypeAdminUpdateCheck:
		return "Admin · Update Check"
	case models.BlockTypeAdminLatestSignups:
		return "Admin · Latest Signups"
	case models.BlockTypeAdminQuickStart:
		return "Admin · Quick Start"
	case models.BlockTypeAdminSystemReload:
		return "Admin · System Reload"
	default:
		return string(t)
	}
}

func blockMatchesListFilters(row models.Block, blockType, query string) bool {
	if blockType != "" && string(row.Type) != blockType {
		return false
	}
	if query == "" {
		return true
	}
	q := strings.ToLower(query)
	for _, field := range []string{
		row.Name,
		string(row.Type),
		blockTypeLabel(row),
		row.ExtensionName,
		row.ExtensionBlockID,
	} {
		if strings.Contains(strings.ToLower(field), q) {
			return true
		}
	}
	return false
}

func sortBlocksByColumn(rows []models.Block, col, dir string) {
	if col == "" {
		col = "sort"
	}
	desc := strings.ToLower(dir) == "desc"
	sort.SliceStable(rows, func(i, j int) bool {
		var cmp int
		switch col {
		case "name":
			cmp = strings.Compare(strings.ToLower(rows[i].Name), strings.ToLower(rows[j].Name))
		case "type":
			cmp = strings.Compare(string(rows[i].Type), string(rows[j].Type))
		case "space":
			cmp = strings.Compare(blocks.PrimarySpace(rows[i]), blocks.PrimarySpace(rows[j]))
		case "status":
			cmp = strings.Compare(string(rows[i].Status), string(rows[j].Status))
		default:
			oi, oj := primarySpaceSort(rows[i]), primarySpaceSort(rows[j])
			cmp = oi - oj
		}
		if cmp == 0 && col != "sort" {
			oi, oj := primarySpaceSort(rows[i]), primarySpaceSort(rows[j])
			cmp = oi - oj
		}
		if cmp == 0 {
			if rows[i].BlockID < rows[j].BlockID {
				cmp = -1
			} else if rows[i].BlockID > rows[j].BlockID {
				cmp = 1
			}
		}
		if desc {
			return cmp > 0
		}
		return cmp < 0
	})
}
