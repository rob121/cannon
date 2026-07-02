package admin

import (
	"context"
	"fmt"
	"net/http"

	"github.com/rob121/cannon/internal/extensions"
	"github.com/rob121/cannon/internal/models"
	"github.com/rob121/cannon/internal/sites"
)

const extensionsBase = "/admin/extensions"

type extensionRow struct {
	models.Extension
	Running bool
	Version string
}

func extensionListRow(row models.Extension, extMgr *extensions.Manager) extensionRow {
	item := extensionRow{
		Extension: row,
		Running:   extMgr.IsRunning(row.Name),
	}
	if meta := extMgr.MetaSummary(row.Name); meta.Available {
		item.Version = meta.Version
	}
	return item
}

func (h *Handler) extensions(w http.ResponseWriter, r *http.Request, path string) {
	parts := pathParts("/extensions", path)
	switch {
	case len(parts) == 0:
		h.extensionList(w, r)
	case len(parts) == 2 && parts[1] == "delete":
		h.extensionDelete(w, r, parts[0])
	case len(parts) == 2 && parts[1] == "restart":
		h.extensionAction(w, r, parts[0], func(ctx context.Context, extMgr *extensions.Manager, row models.Extension) error {
			return extMgr.Restart(ctx, row)
		})
	case len(parts) == 2 && parts[1] == "start":
		h.extensionAction(w, r, parts[0], func(ctx context.Context, extMgr *extensions.Manager, row models.Extension) error {
			return extMgr.Start(ctx, row)
		})
	case len(parts) == 2 && parts[1] == "stop":
		h.extensionAction(w, r, parts[0], func(ctx context.Context, extMgr *extensions.Manager, row models.Extension) error {
			extMgr.Stop(row.Name)
			return nil
		})
	case len(parts) == 2 && parts[1] == "install":
		h.extensionAction(w, r, parts[0], func(ctx context.Context, extMgr *extensions.Manager, row models.Extension) error {
			return extMgr.Install(ctx, row)
		})
	case len(parts) == 2 && parts[1] == "toggle-status":
		h.extensionToggleStatus(w, r, parts[0])
	default:
		id, ok := parseID(parts[0])
		if !ok {
			http.NotFound(w, r)
			return
		}
		h.extensionForm(w, r, id)
	}
}

func (h *Handler) extensionList(w http.ResponseWriter, r *http.Request) {
	db, _ := sites.DB(r.Context())
	site, _ := sites.FromContext(r.Context())
	extMgr := h.chain.Extensions(site)
	_ = extMgr.Bootstrap(r.Context())

	page := parsePage(r)
	var rows []models.Extension
	var total int64
	db.Model(&models.Extension{}).Count(&total)
	data := listPage(page, total, extensionsBase,
		"Installed extension processes and their status.",
		"", map[string]any{"ActiveNav": "extension_registry"})
	order := applyListSort(r, data, map[string]string{
		"name": "name", "sort": "sort", "status": "status", "installed": "installed",
	}, "sort")
	db.Offset((page - 1) * pageSize).Limit(pageSize).Order(order).Find(&rows)
	listRows := make([]extensionRow, 0, len(rows))
	for _, row := range rows {
		listRows = append(listRows, extensionListRow(row, extMgr))
	}
	data["Rows"] = listRows
	h.render(w, r, "Extensions", "admin/extensions.html", data)
}

func (h *Handler) extensionForm(w http.ResponseWriter, r *http.Request, id uint) {
	db, _ := sites.DB(r.Context())
	site, _ := sites.FromContext(r.Context())
	extMgr := h.chain.Extensions(site)

	var row models.Extension
	if err := db.First(&row, id).Error; err != nil {
		http.NotFound(w, r)
		return
	}
	if r.Method == http.MethodPost {
		if err := r.ParseForm(); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		prevStatus := row.Status
		row.MenuName = formString(r, "menu_name")
		row.Sort = formInt(r, "sort", row.Sort)
		row.Status = formStatus(r)
		if err := db.Save(&row).Error; err != nil {
			h.renderExtensionForm(w, r, extMgr, row, err.Error())
			return
		}
		if row.Status == models.StatusActive && prevStatus != models.StatusActive {
			_ = extMgr.Restart(r.Context(), row)
		} else if row.Status == models.StatusInactive && prevStatus == models.StatusActive {
			extMgr.Stop(row.Name)
		} else if r.FormValue("restart") == "1" {
			_ = extMgr.Restart(r.Context(), row)
		}
		redirectList(w, r, extensionsBase)
		return
	}
	h.renderExtensionForm(w, r, extMgr, row, "")
}

func (h *Handler) renderExtensionForm(w http.ResponseWriter, r *http.Request, extMgr *extensions.Manager, row models.Extension, errMsg string) {
	data := formData(map[string]any{
		"ActiveNav":    "extension_registry",
		"Row":          row,
		"Running":      extMgr.IsRunning(row.Name),
		"Meta":         extMgr.MetaSummary(row.Name),
		"Capabilities": extMgr.CapabilitiesSummary(row.Name),
		"BasePath":     extensionsBase,
	})
	if errMsg != "" {
		data["Error"] = errMsg
	}
	h.render(w, r, "Edit Extension", "admin/extensions_form.html", data)
}

func (h *Handler) extensionAction(w http.ResponseWriter, r *http.Request, idStr string, action func(context.Context, *extensions.Manager, models.Extension) error) {
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
	site, _ := sites.FromContext(r.Context())
	extMgr := h.chain.Extensions(site)
	var row models.Extension
	if err := db.First(&row, id).Error; err != nil {
		http.NotFound(w, r)
		return
	}
	if err := action(r.Context(), extMgr, row); err != nil {
		if extensionReturnEdit(r) {
			_ = db.First(&row, id)
			h.renderExtensionForm(w, r, extMgr, row, err.Error())
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	_ = db.First(&row, id)
	extensionAfterAction(w, r, id)
}

func extensionReturnEdit(r *http.Request) bool {
	return r.URL.Query().Get("redirect") == "edit"
}

func extensionEditURL(id uint) string {
	return fmt.Sprintf("%s/%d", extensionsBase, id)
}

func extensionAfterAction(w http.ResponseWriter, r *http.Request, id uint) {
	if extensionReturnEdit(r) {
		redirectList(w, r, extensionEditURL(id))
		return
	}
	redirectList(w, r, extensionsListURL(r))
}

func extensionsListURL(r *http.Request) string {
	if page := r.URL.Query().Get("page"); page != "" && page != "1" {
		return extensionsBase + "?page=" + page
	}
	return extensionsBase
}

func (h *Handler) extensionToggleStatus(w http.ResponseWriter, r *http.Request, idStr string) {
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
	site, _ := sites.FromContext(r.Context())
	extMgr := h.chain.Extensions(site)
	var row models.Extension
	if err := db.First(&row, id).Error; err != nil {
		http.NotFound(w, r)
		return
	}
	prevStatus := row.Status
	row.Status = flipStatus(row.Status)
	if err := db.Save(&row).Error; err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if row.Status == models.StatusActive && prevStatus != models.StatusActive {
		_ = extMgr.Restart(r.Context(), row)
	} else if row.Status == models.StatusInactive && prevStatus == models.StatusActive {
		extMgr.Stop(row.Name)
	}
	redirectList(w, r, extensionsBase+listRedirectQuery(r))
}

func (h *Handler) extensionDelete(w http.ResponseWriter, r *http.Request, idStr string) {
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
	site, _ := sites.FromContext(r.Context())
	var row models.Extension
	if err := db.First(&row, id).Error; err != nil {
		http.NotFound(w, r)
		return
	}
	h.chain.Extensions(site).Stop(row.Name)
	if err := db.Delete(&row).Error; err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	redirectList(w, r, extensionsBase)
}
