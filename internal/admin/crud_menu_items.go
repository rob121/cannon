package admin

import (
	"fmt"
	"net/http"
	"sort"
	"strings"

	"github.com/rob121/cannon/internal/models"
	"github.com/rob121/cannon/internal/router"
	"github.com/rob121/cannon/internal/sites"
	"gorm.io/gorm"
)

const menuItemsBase = "/admin/menu-items"

type menuItemListRow struct {
	Item        models.MenuItem
	Depth       int
	ParentName  string
	CanMoveUp   bool
	CanMoveDown bool
	IsCurrent   bool
}

func (h *Handler) menuItems(w http.ResponseWriter, r *http.Request, path string) {
	parts := pathParts("/menu-items", path)
	switch {
	case len(parts) == 0:
		h.menuItemList(w, r)
	case parts[0] == "new":
		h.menuItemForm(w, r, 0)
	case len(parts) == 2 && parts[1] == "delete":
		h.menuItemDelete(w, r, parts[0])
	case len(parts) == 2 && parts[1] == "move-up":
		h.menuItemMoveSort(w, r, parts[0], -1)
	case len(parts) == 2 && parts[1] == "move-down":
		h.menuItemMoveSort(w, r, parts[0], 1)
	default:
		id, ok := parseID(parts[0])
		if !ok {
			h.notFound(w, r)
			return
		}
		h.menuItemForm(w, r, id)
	}
}

func (h *Handler) menuItemList(w http.ResponseWriter, r *http.Request) {
	db, _ := sites.DB(r.Context())
	menuFilter, hasMenuFilter := parseID(r.URL.Query().Get("menu_id"))

	data := listPage(r, 1, 0, menuItemsBase,
		"Manage links and display order within menus.",
		"Add Menu Item", map[string]any{"ActiveNav": "menu_items"})
	data["MenuFilter"] = menuFilter
	data["HasMenuFilter"] = hasMenuFilter

	order := applyListSort(r, data, map[string]string{
		"name": "name", "menu": "menu_id", "sort": "sort",
	}, "sort")
	sortParam := strings.TrimSpace(r.URL.Query().Get("sort"))
	useTreeOrder := sortParam == "" || sortParam == "sort"

	var rows []models.MenuItem
	q := db.Preload("Route").Preload("Parent")
	if hasMenuFilter {
		q = q.Where("menu_id = ?", menuFilter)
	}
	if useTreeOrder {
		q = q.Order("sort asc, menu_item_id asc")
	} else {
		q = q.Order(order + ", menu_item_id asc")
	}
	q.Find(&rows)
	data["Total"] = int64(len(rows))

	byID := make(map[uint]models.MenuItem, len(rows))
	parentNames := make(map[uint]string, len(rows))
	for _, row := range rows {
		byID[row.MenuItemID] = row
	}
	for _, row := range rows {
		if row.ParentID != nil && *row.ParentID != 0 {
			if parent, ok := byID[*row.ParentID]; ok {
				parentNames[row.MenuItemID] = parent.Name
			}
		}
	}

	positions := menuItemSortPositions(rows)
	depths := menuItemDepthMap(rows)
	treeRows := make([]menuItemListRow, 0, len(rows))
	if useTreeOrder {
		menuIDs := menuItemMenuIDs(rows)
		if hasMenuFilter {
			menuIDs = []uint{menuFilter}
		}
		for _, menuID := range menuIDs {
			menuItems := menuItemsForMenu(rows, menuID)
			treeRows = appendMenuItemTreeRows(treeRows, menuItems, byID, parentNames, positions, 0)
		}
	} else {
		for _, item := range rows {
			pos := positions[item.MenuItemID]
			treeRows = append(treeRows, menuItemListRow{
				Item:        item,
				Depth:       depths[item.MenuItemID],
				ParentName:  parentNames[item.MenuItemID],
				CanMoveUp:   pos.canMoveUp,
				CanMoveDown: pos.canMoveDown,
			})
		}
	}

	data["MenuNames"] = menuNamesByID(db, rows)
	data["AllMenus"] = h.loadMenus(r)
	data["Rows"] = treeRows
	data["ListQuery"] = listQueryFromData(data)
	h.render(w, r, "Menu Items", "admin/menu_items.html", data)
}

func (h *Handler) menuItemForm(w http.ResponseWriter, r *http.Request, id uint) {
	db, _ := sites.DB(r.Context())
	isNew := id == 0
	var row models.MenuItem
	if !isNew {
		if err := db.Preload("Route").Preload("Groups").Preload("Parent").First(&row, id).Error; err != nil {
			h.notFound(w, r)
			return
		}
	} else if menuID, ok := parseID(r.URL.Query().Get("menu_id")); ok {
		row.MenuID = menuID
	}
	if parentID, ok := parseID(r.URL.Query().Get("parent_id")); ok && isNew {
		row.ParentID = &parentID
	}
	allGroups := loadActiveGroups(db)

	if r.Method == http.MethodPost {
		if err := r.ParseForm(); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		row.Name = formString(r, "name")
		row.Class = formString(r, "class")
		row.Target = formString(r, "target")
		row.Sort = formInt(r, "sort", 0)
		row.ParentID = formUintPtr(r, "parent_id")
		if isNew {
			menuID, ok := parseID(formString(r, "menu_id"))
			if !ok {
				h.renderMenuItemForm(w, r, row, allGroups, isNew, "Select a menu.")
				return
			}
			row.MenuID = menuID
		}
		menuItems, err := loadMenuItemsForMenu(db, row.MenuID)
		if err != nil {
			h.renderMenuItemForm(w, r, row, allGroups, isNew, err.Error())
			return
		}
		if err := router.ValidateMenuItemParent(menuItems, row, row.ParentID); err != nil {
			h.renderMenuItemForm(w, r, row, allGroups, isNew, "Select a valid parent item in the same menu.")
			return
		}
		if rid := formString(r, "route_id"); rid != "" {
			if routeID, ok := parseID(rid); ok {
				row.RouteID = &routeID
			}
		} else {
			row.RouteID = nil
		}
		var saveErr error
		if isNew {
			saveErr = db.Create(&row).Error
		} else {
			saveErr = db.Save(&row).Error
		}
		if saveErr != nil {
			h.renderMenuItemForm(w, r, row, allGroups, isNew, saveErr.Error())
			return
		}
		if err := replaceFormGroups(db, &row, r); err != nil {
			h.renderMenuItemForm(w, r, row, allGroups, isNew, err.Error())
			return
		}
		redirectList(w, r, menuItemsBase)
		return
	}
	h.renderMenuItemForm(w, r, row, allGroups, isNew, "")
}

func (h *Handler) renderMenuItemForm(w http.ResponseWriter, r *http.Request, row models.MenuItem, allGroups []models.Group, isNew bool, errMsg string) {
	title := "Add Menu Item"
	subtitle := "Create a new entry in a navigation menu."
	if !isNew {
		title = "Edit Menu Item"
		subtitle = "Configure label, linked route, and display order."
	}
	db, _ := sites.DB(r.Context())
	menuName := menuNameForID(db, row.MenuID)
	parentOptions, _ := allMenuItemParentOptions(db, row.MenuItemID)
	parentName := ""
	if row.ParentID != nil && *row.ParentID != 0 {
		var parent models.MenuItem
		if err := db.Select("name").First(&parent, *row.ParentID).Error; err == nil {
			parentName = parent.Name
		}
	}
	data := formData(map[string]any{
		"ActiveNav":     "menu_items",
		"Row":           row,
		"MenuName":      menuName,
		"ParentName":    parentName,
		"ParentOptions": parentOptions,
		"IsNew":         isNew,
		"BasePath":      menuItemsBase,
		"Subtitle":      subtitle,
		"Routes":        h.loadRoutes(r),
		"AllMenus":      h.loadMenus(r),
		"AllGroups":     allGroups,
		"SelectedIDs":   defaultGroupSelectedIDs(db, row.Groups, isNew),
	})
	if errMsg != "" {
		data["Error"] = errMsg
	}
	if !isNew && row.MenuID != 0 {
		if items, err := loadMenuItemsForMenu(db, row.MenuID); err == nil {
			data["MenuItemRows"] = buildMenuItemListRows(items, row.MenuItemID)
			data["MenuItemsURL"] = fmt.Sprintf("%s?menu_id=%d", menuItemsBase, row.MenuID)
		}
	}
	data["ListQuery"] = menuItemListQuery(r)
	h.render(w, r, title, "admin/menu_item_form.html", data)
}

func (h *Handler) menuItemDelete(w http.ResponseWriter, r *http.Request, idStr string) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	id, ok := parseID(idStr)
	if !ok {
		h.notFound(w, r)
		return
	}
	db, _ := sites.DB(r.Context())
	var row models.MenuItem
	if err := db.First(&row, id).Error; err != nil {
		h.notFound(w, r)
		return
	}
	if err := db.Model(&models.MenuItem{}).Where("parent_id = ?", id).Update("parent_id", row.ParentID).Error; err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if err := db.Exec("DELETE FROM menu_item_groups WHERE menu_item_menu_item_id = ?", id).Error; err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if err := db.Delete(&models.MenuItem{}, id).Error; err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	redirectList(w, r, menuItemsBase)
}

func (h *Handler) loadMenus(r *http.Request) []models.Menu {
	db, _ := sites.DB(r.Context())
	var menus []models.Menu
	db.Order("menu_name asc").Find(&menus)
	return menus
}

func (h *Handler) loadRoutes(r *http.Request) []models.Route {
	db, _ := sites.DB(r.Context())
	var routes []models.Route
	db.Where("status = ?", models.StatusActive).Order("name asc").Find(&routes)
	return routes
}

func menuItemListQuery(r *http.Request) string {
	q := r.URL.Query()
	sortParam := strings.TrimSpace(q.Get("sort"))
	dir := strings.TrimSpace(q.Get("dir"))
	if sortParam == "" && dir == "" && q.Get("menu_id") == "" {
		return ""
	}
	extra := listExtraFromData(nil)
	if menuID := strings.TrimSpace(q.Get("menu_id")); menuID != "" {
		extra.Set("menu_id", menuID)
	}
	return listQueryExtra(1, sortParam, dir, extra)
}

func buildMenuItemListRows(items []models.MenuItem, currentID uint) []menuItemListRow {
	byID := make(map[uint]models.MenuItem, len(items))
	parentNames := make(map[uint]string, len(items))
	for _, item := range items {
		byID[item.MenuItemID] = item
	}
	for _, item := range items {
		if item.ParentID != nil && *item.ParentID != 0 {
			if parent, ok := byID[*item.ParentID]; ok {
				parentNames[item.MenuItemID] = parent.Name
			}
		}
	}
	treeRows := make([]menuItemListRow, 0, len(items))
	return appendMenuItemTreeRows(treeRows, items, byID, parentNames, nil, currentID)
}

func appendMenuItemTreeRows(out []menuItemListRow, items []models.MenuItem, byID map[uint]models.MenuItem, parentNames map[uint]string, positions map[uint]menuItemSortPosition, currentID uint) []menuItemListRow {
	for _, opt := range router.FlattenMenuItemsForList(items) {
		item, ok := byID[opt.MenuItemID]
		if !ok {
			continue
		}
		row := menuItemListRow{
			Item:       item,
			Depth:      opt.Depth,
			ParentName: parentNames[item.MenuItemID],
			IsCurrent:  currentID != 0 && item.MenuItemID == currentID,
		}
		if positions != nil {
			pos := positions[item.MenuItemID]
			row.CanMoveUp = pos.canMoveUp
			row.CanMoveDown = pos.canMoveDown
		}
		out = append(out, row)
	}
	return out
}

func menuItemsForMenu(items []models.MenuItem, menuID uint) []models.MenuItem {
	out := make([]models.MenuItem, 0)
	for _, item := range items {
		if item.MenuID == menuID {
			out = append(out, item)
		}
	}
	return out
}

func menuItemMenuIDs(items []models.MenuItem) []uint {
	seen := make(map[uint]struct{})
	out := make([]uint, 0)
	for _, item := range items {
		if item.MenuID == 0 {
			continue
		}
		if _, ok := seen[item.MenuID]; ok {
			continue
		}
		seen[item.MenuID] = struct{}{}
		out = append(out, item.MenuID)
	}
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}

func menuItemDepthMap(items []models.MenuItem) map[uint]int {
	byID := make(map[uint]models.MenuItem, len(items))
	for _, item := range items {
		byID[item.MenuItemID] = item
	}
	depths := make(map[uint]int, len(items))
	var depthOf func(id uint) int
	depthOf = func(id uint) int {
		if d, ok := depths[id]; ok {
			return d
		}
		item, ok := byID[id]
		if !ok {
			depths[id] = 0
			return 0
		}
		if item.ParentID == nil || *item.ParentID == 0 {
			depths[id] = 0
			return 0
		}
		if _, ok := byID[*item.ParentID]; !ok {
			depths[id] = 0
			return 0
		}
		d := depthOf(*item.ParentID) + 1
		depths[id] = d
		return d
	}
	for _, item := range items {
		depthOf(item.MenuItemID)
	}
	return depths
}

func allMenuItemParentOptions(db *gorm.DB, excludeID uint) ([]router.MenuItemParentOption, error) {
	var items []models.MenuItem
	if err := db.Order("menu_id asc, sort asc, menu_item_id asc").Find(&items).Error; err != nil {
		return nil, err
	}
	byMenu := make(map[uint][]models.MenuItem)
	menuIDs := make([]uint, 0)
	seenMenu := make(map[uint]struct{})
	for _, item := range items {
		byMenu[item.MenuID] = append(byMenu[item.MenuID], item)
		if _, ok := seenMenu[item.MenuID]; ok {
			continue
		}
		seenMenu[item.MenuID] = struct{}{}
		menuIDs = append(menuIDs, item.MenuID)
	}
	sort.Slice(menuIDs, func(i, j int) bool { return menuIDs[i] < menuIDs[j] })
	out := make([]router.MenuItemParentOption, 0, len(items))
	for _, menuID := range menuIDs {
		out = append(out, router.MenuItemParentOptions(byMenu[menuID], menuID, excludeID)...)
	}
	return out, nil
}

func loadMenuItemsForMenu(db *gorm.DB, menuID uint) ([]models.MenuItem, error) {
	var items []models.MenuItem
	if err := db.Where("menu_id = ?", menuID).Preload("Route").Order("sort asc, menu_item_id asc").Find(&items).Error; err != nil {
		return nil, err
	}
	return items, nil
}

func menuNameForID(db *gorm.DB, menuID uint) string {
	if menuID == 0 {
		return ""
	}
	var menu models.Menu
	if err := db.Select("menu_name").First(&menu, menuID).Error; err != nil {
		return ""
	}
	return menu.MenuName
}

func menuNamesByID(db *gorm.DB, rows []models.MenuItem) map[uint]string {
	names := make(map[uint]string)
	if len(rows) == 0 {
		return names
	}
	ids := make([]uint, 0, len(rows))
	seen := make(map[uint]struct{})
	for _, row := range rows {
		if row.MenuID == 0 {
			continue
		}
		if _, ok := seen[row.MenuID]; ok {
			continue
		}
		seen[row.MenuID] = struct{}{}
		ids = append(ids, row.MenuID)
	}
	if len(ids) == 0 {
		return names
	}
	var menus []models.Menu
	if err := db.Select("menu_id", "menu_name").Where("menu_id IN ?", ids).Find(&menus).Error; err != nil {
		return names
	}
	for _, menu := range menus {
		names[menu.MenuID] = menu.MenuName
	}
	return names
}

type menuItemSortPosition struct {
	canMoveUp   bool
	canMoveDown bool
}

func menuItemSiblingKey(row models.MenuItem) string {
	parent := uint(0)
	if row.ParentID != nil {
		parent = *row.ParentID
	}
	return fmt.Sprintf("%d:%d", row.MenuID, parent)
}

func menuItemSortPositions(items []models.MenuItem) map[uint]menuItemSortPosition {
	byGroup := make(map[string][]models.MenuItem)
	for _, row := range items {
		key := menuItemSiblingKey(row)
		byGroup[key] = append(byGroup[key], row)
	}
	out := make(map[uint]menuItemSortPosition, len(items))
	for _, group := range byGroup {
		sort.Slice(group, func(i, j int) bool {
			if group[i].Sort != group[j].Sort {
				return group[i].Sort < group[j].Sort
			}
			return group[i].MenuItemID < group[j].MenuItemID
		})
		last := len(group) - 1
		for i, row := range group {
			out[row.MenuItemID] = menuItemSortPosition{
				canMoveUp:   i > 0,
				canMoveDown: i < last,
			}
		}
	}
	return out
}

func (h *Handler) menuItemMoveSort(w http.ResponseWriter, r *http.Request, idStr string, direction int) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	id, ok := parseID(idStr)
	if !ok {
		h.notFound(w, r)
		return
	}
	db, _ := sites.DB(r.Context())
	if err := menuItemReorder(db, id, direction); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	redirectList(w, r, menuItemsBase+listRedirectQuery(r))
}

func menuItemReorder(db *gorm.DB, id uint, direction int) error {
	if direction == 0 {
		return nil
	}
	var row models.MenuItem
	if err := db.First(&row, id).Error; err != nil {
		return err
	}
	var siblings []models.MenuItem
	q := db.Where("menu_id = ?", row.MenuID)
	if row.ParentID == nil || *row.ParentID == 0 {
		q = q.Where("parent_id IS NULL OR parent_id = 0")
	} else {
		q = q.Where("parent_id = ?", *row.ParentID)
	}
	if err := q.Order("sort asc, menu_item_id asc").Find(&siblings).Error; err != nil {
		return err
	}
	idx := -1
	for i, item := range siblings {
		if item.MenuItemID == id {
			idx = i
			break
		}
	}
	if idx < 0 {
		return gorm.ErrRecordNotFound
	}
	target := idx + direction
	if target < 0 || target >= len(siblings) {
		return nil
	}
	siblings[idx], siblings[target] = siblings[target], siblings[idx]
	for i, item := range siblings {
		if item.Sort == i {
			continue
		}
		if err := db.Model(&models.MenuItem{}).Where("menu_item_id = ?", item.MenuItemID).Update("sort", i).Error; err != nil {
			return err
		}
	}
	return nil
}
