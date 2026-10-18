package notifications

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/rob121/cannon/internal/models"
	"gorm.io/gorm"
)

// SubscriptionFilters controls when a Layer 2 subscription matches an event.
type SubscriptionFilters struct {
	Status     string `json:"status,omitempty"`      // item status must equal (e.g. pending)
	Category   string `json:"category,omitempty"`    // category id or slug
	AuthorSelf bool   `json:"author_self,omitempty"` // item author matches recipient
}

// ParseFiltersJSON decodes subscription filters.
func ParseFiltersJSON(raw string) SubscriptionFilters {
	raw = strings.TrimSpace(raw)
	if raw == "" || raw == "{}" {
		return SubscriptionFilters{}
	}
	var out SubscriptionFilters
	_ = json.Unmarshal([]byte(raw), &out)
	out.Status = strings.TrimSpace(out.Status)
	out.Category = strings.TrimSpace(out.Category)
	return out
}

// FiltersToJSON encodes filters for storage.
func FiltersToJSON(f SubscriptionFilters) string {
	f.Status = strings.TrimSpace(f.Status)
	f.Category = strings.TrimSpace(f.Category)
	if f.Status == "" && f.Category == "" && !f.AuthorSelf {
		return ""
	}
	raw, err := json.Marshal(f)
	if err != nil {
		return ""
	}
	return string(raw)
}

// MatchFilters reports whether filters allow delivery for this recipient.
func MatchFilters(db *gorm.DB, filtersJSON string, args map[string]any, recipientUserID uint) bool {
	filters := ParseFiltersJSON(filtersJSON)
	if filters.Status == "" && filters.Category == "" && !filters.AuthorSelf {
		return true
	}
	vars := FlattenArgs(args)
	if filters.Status != "" {
		status := vars["status"]
		if status == "" {
			status = itemStatusFromArgs(args)
		}
		if !strings.EqualFold(status, filters.Status) {
			return false
		}
	}
	if filters.Category != "" {
		if !categoryMatches(db, filters.Category, vars, args) {
			return false
		}
	}
	if filters.AuthorSelf {
		authorID := vars["author_id"]
		if authorID == "" {
			authorID = itemAuthorFromArgs(args)
		}
		if authorID == "" || fmt.Sprint(recipientUserID) != authorID {
			return false
		}
	}
	return true
}

func itemStatusFromArgs(args map[string]any) string {
	item := itemFromArgs(args)
	if item == nil {
		return ""
	}
	return string(item.Status)
}

func itemAuthorFromArgs(args map[string]any) string {
	item := itemFromArgs(args)
	if item == nil || item.AuthorID == nil {
		return ""
	}
	return fmt.Sprint(*item.AuthorID)
}

func itemFromArgs(args map[string]any) *models.Item {
	if args == nil {
		return nil
	}
	switch item := args["item"].(type) {
	case models.Item:
		return &item
	case *models.Item:
		return item
	default:
		return nil
	}
}

func categoryMatches(db *gorm.DB, want string, vars map[string]string, args map[string]any) bool {
	want = strings.TrimSpace(want)
	if want == "" {
		return true
	}
	if strings.EqualFold(vars["category_id"], want) || strings.EqualFold(vars["category_slug"], want) || strings.EqualFold(vars["category"], want) {
		return true
	}
	item := itemFromArgs(args)
	if item != nil && item.CategoryID != nil {
		if strings.EqualFold(fmt.Sprint(*item.CategoryID), want) {
			return true
		}
		if id, err := strconv.ParseUint(want, 10, 64); err == nil && uint(id) == *item.CategoryID {
			return true
		}
	}
	if db == nil {
		return false
	}
	// Resolve slug → id when item only has category_id.
	if item != nil && item.CategoryID != nil {
		var cat models.Category
		if err := db.Select("category_id", "slug").First(&cat, *item.CategoryID).Error; err == nil {
			if strings.EqualFold(cat.Slug, want) || fmt.Sprint(cat.CategoryID) == want {
				return true
			}
		}
	}
	if itemID := strings.TrimSpace(vars["item_id"]); itemID != "" && item == nil {
		var row models.Item
		if err := db.Select("category_id").First(&row, itemID).Error; err == nil && row.CategoryID != nil {
			if strings.EqualFold(fmt.Sprint(*row.CategoryID), want) {
				return true
			}
			var cat models.Category
			if err := db.Select("slug").First(&cat, *row.CategoryID).Error; err == nil && strings.EqualFold(cat.Slug, want) {
				return true
			}
		}
	}
	return false
}
