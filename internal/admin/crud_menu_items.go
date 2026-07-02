package admin

import (
	"net/http"

	"github.com/rob121/cannon/internal/models"
	"github.com/rob121/cannon/internal/sites"
	"gorm.io/gorm"
)

const menuItemsBase = "/admin/menu-items"

func (h *Handler) menuItems(w http.ResponseWriter, r *http.Request, path string) {
	parts := pathParts("/menu-items", path)
	switch {
	case len(parts) == 0:
		h.menuItemList(w, r)
	case parts[0] == "new":
		h.menuItemForm(w, r, 0)
	case len(parts) == 2 && parts[1] == "delete":
		h.menuItemDelete(w, r, parts[0])
	default:
		id, ok := parseID(parts[0])
		if !ok {
			http.NotFound(w, r)
			return
		}
		h.menuItemForm(w, r, id)
	}
}

func (h *Handler) menuItemList(w http.ResponseWriter, r *http.Request) {
	db, _ := sites.DB(r.Context())
	page := parsePage(r)
	var rows []models.MenuItem
	var total int64
	db.Model(&models.MenuItem{}).Count(&total)
	data := listPage(page, total, menuItemsBase,
		"Manage links and display order within menus.",
		"Add Menu Item", map[string]any{"ActiveNav": "menu_items"})
	order := applyListSort(r, data, map[string]string{
		"name": "name", "sort": "sort",
	}, "sort")
	db.Offset((page - 1) * pageSize).Limit(pageSize).
		Preload("Route").
		Order(order).
		Find(&rows)
	menuNames := menuNamesByID(db, rows)
	data["MenuNames"] = menuNames
	data["Rows"] = rows
	h.render(w, r, "Menu Items", "admin/menu_items.html", data)
}

func (h *Handler) menuItemForm(w http.ResponseWriter, r *http.Request, id uint) {
	db, _ := sites.DB(r.Context())
	isNew := id == 0
	var row models.MenuItem
	if !isNew {
		if err := db.Preload("Route").Preload("Groups").First(&row, id).Error; err != nil {
			http.NotFound(w, r)
			return
		}
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
		if isNew {
			menuID, ok := parseID(formString(r, "menu_id"))
			if !ok {
				h.renderMenuItemForm(w, r, row, allGroups, isNew, "Select a menu.")
				return
			}
			row.MenuID = menuID
		}
		if rid := formString(r, "route_id"); rid != "" {
			if routeID, ok := parseID(rid); ok {
				row.RouteID = &routeID
			}
		} else {
			row.RouteID = nil
		}
		var err error
		if isNew {
			err = db.Create(&row).Error
		} else {
			err = db.Save(&row).Error
		}
		if err != nil {
			h.renderMenuItemForm(w, r, row, allGroups, isNew, err.Error())
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
	data := formData(map[string]any{
		"ActiveNav":   "menu_items",
		"Row":         row,
		"MenuName":    menuName,
		"IsNew":       isNew,
		"BasePath":    menuItemsBase,
		"Subtitle":    subtitle,
		"Routes":      h.loadRoutes(r),
		"AllMenus":    h.loadMenus(r),
		"AllGroups":   allGroups,
		"SelectedIDs": defaultGroupSelectedIDs(db, row.Groups, isNew),
	})
	if errMsg != "" {
		data["Error"] = errMsg
	}
	h.render(w, r, title, "admin/menu_item_form.html", data)
}

func (h *Handler) menuItemDelete(w http.ResponseWriter, r *http.Request, idStr string) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	id, ok := parseID(idStr)
	if !ok {
		http.NotFound(w, r)
		return
	}
	db, _ := sites.DB(r.Context())
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
