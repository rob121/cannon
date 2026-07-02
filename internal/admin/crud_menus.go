package admin

import (
	"fmt"
	"net/http"

	"github.com/rob121/cannon/internal/models"
	"github.com/rob121/cannon/internal/sites"
	"gorm.io/gorm"
)

const menusBase = "/admin/menus"

func (h *Handler) menus(w http.ResponseWriter, r *http.Request, path string) {
	parts := pathParts("/menus", path)
	switch {
	case len(parts) == 0:
		h.menuList(w, r)
	case parts[0] == "new":
		h.menuForm(w, r, 0)
	case len(parts) == 2 && parts[1] == "delete":
		h.menuDelete(w, r, parts[0])
	case len(parts) == 2 && parts[1] == "toggle-status":
		h.menuToggleStatus(w, r, parts[0])
	default:
		id, ok := parseID(parts[0])
		if !ok {
			h.notFound(w, r)
			return
		}
		h.menuForm(w, r, id)
	}
}

func (h *Handler) menuList(w http.ResponseWriter, r *http.Request) {
	db, _ := sites.DB(r.Context())
	page := parsePage(r)
	var rows []models.Menu
	var total int64
	db.Model(&models.Menu{}).Count(&total)
	data := listPage(r, page, total, menusBase,
		"Manage navigation menu definitions.",
		"Add Menu", map[string]any{"ActiveNav": "menus"})
	order := applyListSort(r, data, map[string]string{
		"name": "menu_name", "status": "status",
	}, "name")
	db.Offset((page - 1) * pageSizeFor(r)).Limit(pageSizeFor(r)).Preload("Items").Order(order).Find(&rows)
	data["Rows"] = rows
	h.render(w, r, "Menus", "admin/menus.html", data)
}

func (h *Handler) menuForm(w http.ResponseWriter, r *http.Request, id uint) {
	db, _ := sites.DB(r.Context())
	isNew := id == 0
	var row models.Menu
	if !isNew {
		if err := db.Preload("Items", func(db *gorm.DB) *gorm.DB {
			return db.Order("sort asc, menu_item_id asc")
		}).Preload("Items.Route").First(&row, id).Error; err != nil {
			h.notFound(w, r)
			return
		}
	}
	if r.Method == http.MethodPost {
		if err := r.ParseForm(); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		row.MenuName = formString(r, "menu_name")
		row.Status = formStatus(r)
		var err error
		if isNew {
			err = db.Create(&row).Error
		} else {
			err = db.Save(&row).Error
		}
		if err != nil {
			h.renderMenuForm(w, r, row, isNew, err.Error())
			return
		}
		redirectList(w, r, menusBase)
		return
	}
	h.renderMenuForm(w, r, row, isNew, "")
}

func (h *Handler) renderMenuForm(w http.ResponseWriter, r *http.Request, row models.Menu, isNew bool, errMsg string) {
	title := "Add Menu"
	subtitle := "Create a named menu container for navigation items."
	if !isNew {
		title = "Edit Menu"
		subtitle = "Update the menu name and visibility status."
	}
	data := h.menuFormData(row, isNew, subtitle, errMsg)
	if !isNew {
		data["ItemRows"] = buildMenuItemListRows(row.Items, 0)
		data["MenuItemsURL"] = fmt.Sprintf("%s?menu_id=%d", menuItemsBase, row.MenuID)
		data["AddMenuItemURL"] = fmt.Sprintf("%s/new?menu_id=%d", menuItemsBase, row.MenuID)
	}
	h.render(w, r, title, "admin/menus_form.html", data)
}

func (h *Handler) menuFormData(row models.Menu, isNew bool, subtitle, errMsg string) map[string]any {
	data := formData(map[string]any{
		"ActiveNav": "menus",
		"Row":       row,
		"IsNew":     isNew,
		"BasePath":  menusBase,
		"Subtitle":  subtitle,
	})
	if errMsg != "" {
		data["Error"] = errMsg
	}
	return data
}

func (h *Handler) menuDelete(w http.ResponseWriter, r *http.Request, idStr string) {
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
	db.Where("menu_id = ?", id).Delete(&models.MenuItem{})
	if err := db.Delete(&models.Menu{}, id).Error; err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	redirectList(w, r, menusBase)
}
