package router

import (
	"context"
	"errors"
	"sort"
	"strings"

	"github.com/rob121/cannon/internal/groups"
	"github.com/rob121/cannon/internal/models"
	"github.com/rob121/cannon/internal/sites"
	"gorm.io/gorm"
)

var errInvalidMenuItemParent = errors.New("invalid menu item parent")

// MenuItemParentOption is a selectable parent for admin menu item forms.
type MenuItemParentOption struct {
	MenuItemID uint
	MenuID     uint
	Name       string
	Label      string
	Depth      int
}

// MenuData loads a hierarchical menu for template rendering.
// Top-level items include a "Children" slice of nested item maps when present.
func MenuData(ctx context.Context, menuName string) ([]map[string]any, error) {
	items, err := loadActiveMenuItems(ctx, menuName)
	if err != nil {
		return nil, err
	}
	return buildMenuTree(items), nil
}

// MenuDataWithDepth loads a menu tree limited to maxDepth levels (1 = roots only).
func MenuDataWithDepth(ctx context.Context, menuName string, maxDepth int) ([]map[string]any, error) {
	items, err := MenuData(ctx, menuName)
	if err != nil {
		return nil, err
	}
	if maxDepth > 0 {
		items = LimitMenuDepth(items, maxDepth)
	}
	return items, nil
}

// LimitMenuDepth trims nested Children beyond maxDepth levels.
func LimitMenuDepth(items []map[string]any, maxDepth int) []map[string]any {
	return limitMenuItems(items, maxDepth, 1)
}

func limitMenuItems(items []map[string]any, maxDepth, depth int) []map[string]any {
	if len(items) == 0 {
		return nil
	}
	out := make([]map[string]any, 0, len(items))
	for _, item := range items {
		out = append(out, limitMenuItem(item, maxDepth, depth))
	}
	return out
}

func limitMenuItem(item map[string]any, maxDepth, depth int) map[string]any {
	view := map[string]any{
		"MenuItemID": item["MenuItemID"],
		"Name":       item["Name"],
		"Href":       item["Href"],
		"Class":      item["Class"],
		"Target":     item["Target"],
	}
	if depth >= maxDepth {
		return view
	}
	children, _ := item["Children"].([]map[string]any)
	if len(children) > 0 {
		view["Children"] = limitMenuItems(children, maxDepth, depth+1)
	}
	return view
}

func loadActiveMenuItems(ctx context.Context, menuName string) ([]models.MenuItem, error) {
	db, err := sites.DB(ctx)
	if err != nil {
		return nil, err
	}
	viewerGroups, err := groups.ViewerGroupIDs(ctx)
	if err != nil {
		return nil, err
	}

	menuName = strings.TrimSpace(menuName)
	var menu models.Menu
	if err := db.Preload("Items", func(db *gorm.DB) *gorm.DB {
		return db.Order("sort asc, menu_item_id asc")
	}).Preload("Items.Groups").Preload("Items.Route").
		Where("LOWER(menu_name) = LOWER(?) AND status = ?", menuName, models.StatusActive).
		First(&menu).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return []models.MenuItem{}, nil
		}
		return nil, err
	}

	viewable := make([]models.MenuItem, 0, len(menu.Items))
	for _, item := range menu.Items {
		if groups.CanView(viewerGroups, item.Groups) {
			viewable = append(viewable, item)
		}
	}
	return viewable, nil
}

func buildMenuTree(items []models.MenuItem) []map[string]any {
	if len(items) == 0 {
		return []map[string]any{}
	}
	byID := make(map[uint]models.MenuItem, len(items))
	children := make(map[uint][]models.MenuItem)
	roots := make([]models.MenuItem, 0, len(items))
	for _, item := range items {
		byID[item.MenuItemID] = item
	}
	for _, item := range items {
		if item.ParentID == nil || *item.ParentID == 0 {
			roots = append(roots, item)
			continue
		}
		if _, ok := byID[*item.ParentID]; !ok {
			roots = append(roots, item)
			continue
		}
		children[*item.ParentID] = append(children[*item.ParentID], item)
	}
	sortMenuItems(roots)
	for id := range children {
		sortMenuItems(children[id])
	}
	out := make([]map[string]any, 0, len(roots))
	for _, item := range roots {
		out = append(out, menuItemView(item, children))
	}
	return out
}

func menuItemView(item models.MenuItem, children map[uint][]models.MenuItem) map[string]any {
	view := map[string]any{
		"MenuItemID": item.MenuItemID,
		"Name":       item.Name,
		"Href":       menuItemHref(item),
		"Class":      item.Class,
		"Target":     item.Target,
	}
	kids := children[item.MenuItemID]
	if len(kids) == 0 {
		return view
	}
	nested := make([]map[string]any, 0, len(kids))
	for _, child := range kids {
		nested = append(nested, menuItemView(child, children))
	}
	view["Children"] = nested
	return view
}

func menuItemHref(item models.MenuItem) string {
	if item.Route != nil && item.Route.Status == models.StatusActive {
		return item.Route.Path
	}
	return "#"
}

func sortMenuItems(items []models.MenuItem) {
	sort.SliceStable(items, func(i, j int) bool {
		if items[i].Sort != items[j].Sort {
			return items[i].Sort < items[j].Sort
		}
		return items[i].MenuItemID < items[j].MenuItemID
	})
}

// MenuItemParentOptions returns valid parent choices for a menu item form.
func MenuItemParentOptions(items []models.MenuItem, menuID, excludeID uint) []MenuItemParentOption {
	filtered := make([]models.MenuItem, 0, len(items))
	for _, item := range items {
		if item.MenuID != menuID {
			continue
		}
		if excludeID != 0 && item.MenuItemID == excludeID {
			continue
		}
		if excludeID != 0 && menuItemIsDescendant(items, excludeID, item.MenuItemID) {
			continue
		}
		filtered = append(filtered, item)
	}
	opts := flattenMenuItemOptions(buildMenuTree(filtered), 0)
	for i := range opts {
		opts[i].MenuID = menuID
	}
	return opts
}

func flattenMenuItemOptions(items []map[string]any, depth int) []MenuItemParentOption {
	out := make([]MenuItemParentOption, 0, len(items))
	for _, item := range items {
		id, _ := item["MenuItemID"].(uint)
		name, _ := item["Name"].(string)
		out = append(out, MenuItemParentOption{
			MenuItemID: id,
			Name:       name,
			Label:      strings.Repeat("— ", depth) + name,
			Depth:      depth,
		})
		kids, _ := item["Children"].([]map[string]any)
		out = append(out, flattenMenuItemOptions(kids, depth+1)...)
	}
	return out
}

// FlattenMenuItemsForList returns menu items in tree order for admin tables.
func FlattenMenuItemsForList(items []models.MenuItem) []MenuItemParentOption {
	return flattenMenuItemOptions(buildMenuTree(items), 0)
}

func menuItemIsDescendant(items []models.MenuItem, ancestorID, candidateID uint) bool {
	byID := make(map[uint]models.MenuItem, len(items))
	for _, item := range items {
		byID[item.MenuItemID] = item
	}
	current, ok := byID[candidateID]
	for ok {
		if current.ParentID == nil || *current.ParentID == 0 {
			return false
		}
		if *current.ParentID == ancestorID {
			return true
		}
		current, ok = byID[*current.ParentID]
	}
	return false
}

// ValidateMenuItemParent ensures a parent belongs to the same menu and does not create a cycle.
func ValidateMenuItemParent(items []models.MenuItem, row models.MenuItem, parentID *uint) error {
	if parentID == nil || *parentID == 0 {
		return nil
	}
	var parent models.MenuItem
	found := false
	for _, item := range items {
		if item.MenuItemID == *parentID {
			parent = item
			found = true
			break
		}
	}
	if !found {
		return errInvalidMenuItemParent
	}
	if parent.MenuID != row.MenuID {
		return errInvalidMenuItemParent
	}
	if row.MenuItemID != 0 && (*parentID == row.MenuItemID || menuItemIsDescendant(items, row.MenuItemID, *parentID)) {
		return errInvalidMenuItemParent
	}
	return nil
}
