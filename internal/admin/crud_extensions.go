package admin

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/rob121/cannon/internal/extensions"
	"github.com/rob121/cannon/internal/models"
	"github.com/rob121/cannon/internal/sites"
	"github.com/rob121/cannon/internal/templatemgr"
	"gorm.io/gorm"
)

const extensionsBase = "/admin/extensions"

type extensionRow struct {
	models.Extension
	Running      bool
	Version      string
	DisplayTitle string
	Description  string
	CanMoveUp    bool
	CanMoveDown  bool
}

func extensionListRow(row models.Extension, extMgr *extensions.Manager) extensionRow {
	item := extensionRow{
		Extension: row,
		Running:   extMgr.IsRunning(row.Name),
	}
	if meta := extMgr.MetaSummary(row.Name); meta.Available {
		item.Version = meta.Version
		item.DisplayTitle = meta.Title
		item.Description = meta.Description
	} else {
		item.Version = row.Version
		item.DisplayTitle = row.Title
		item.Description = row.Description
	}
	item.DisplayTitle = extensionDisplayTitle(item.DisplayTitle, row.MenuName, row.Name)
	return item
}

func extensionDisplayTitle(title, menuName, binaryName string) string {
	if v := strings.TrimSpace(title); v != "" {
		return v
	}
	if v := strings.TrimSpace(menuName); v != "" {
		return v
	}
	return binaryName
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
	case len(parts) == 2 && parts[1] == "move-up":
		h.extensionMoveSort(w, r, parts[0], -1)
	case len(parts) == 2 && parts[1] == "move-down":
		h.extensionMoveSort(w, r, parts[0], 1)
	case len(parts) == 2 && parts[1] == "toggle-status":
		h.extensionToggleStatus(w, r, parts[0])
	case len(parts) == 3 && parts[1] == "templates" && parts[2] == "override":
		h.extensionTemplateOverride(w, r, parts[0])
	default:
		id, ok := parseID(parts[0])
		if !ok {
			h.notFound(w, r)
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
	data := listPage(r, page, total, extensionsBase,
		"Installed extension processes and their status.",
		"", map[string]any{"ActiveNav": "extension_registry"})
	order := applyListSort(r, data, map[string]string{
		"name": "name", "status": "status", "installed": "installed", "sort": "sort",
	}, "sort")
	sortParam := strings.TrimSpace(r.URL.Query().Get("sort"))
	if sortParam == "" || sortParam == "sort" {
		order = "sort asc, extension_id asc"
	}
	db.Offset((page - 1) * pageSizeFor(r)).Limit(pageSizeFor(r)).Order(order).Find(&rows)

	var ordered []models.Extension
	db.Order("sort asc, extension_id asc").Find(&ordered)
	position := make(map[uint]int, len(ordered))
	for i, row := range ordered {
		position[row.ExtensionID] = i
	}
	last := len(ordered) - 1

	listRows := make([]extensionRow, 0, len(rows))
	for _, row := range rows {
		item := extensionListRow(row, extMgr)
		if pos, ok := position[row.ExtensionID]; ok {
			item.CanMoveUp = pos > 0
			item.CanMoveDown = pos < last
		}
		listRows = append(listRows, item)
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
		h.notFound(w, r)
		return
	}
	if r.Method == http.MethodPost {
		if err := r.ParseForm(); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		prevStatus := row.Status
		row.MenuName = formString(r, "menu_name")
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
	meta := extMgr.MetaSummary(row.Name)
	site, _ := sites.FromContext(r.Context())
	templateDir := ""
	if site != nil {
		templateDir = site.TemplateDir
	}
	displayTitle := extensionDisplayTitle(firstNonEmpty(meta.Title, row.Title), row.MenuName, row.Name)
	subtitle := row.Name
	if version := firstNonEmpty(meta.Version, row.Version); version != "" {
		subtitle += " · v" + version
	}
	hasConfig := false
	configURL := ""
	if rt, ok := extMgr.Runtime(row.Name); ok && strings.TrimSpace(rt.Capabilities.Configuration) != "" {
		hasConfig = true
		configURL = configurationBase + "/extensions/" + url.PathEscape(row.Name)
	}
	data := formData(map[string]any{
		"ActiveNav":        "extension_registry",
		"Row":              row,
		"Running":          extMgr.IsRunning(row.Name),
		"Meta":             mergeExtensionMeta(meta, row),
		"Capabilities":     extMgr.CapabilitiesSummary(row.Name),
		"Templates":        extMgr.TemplateSummary(row.Name, templateDir),
		"BasePath":         extensionsBase,
		"TemplatesBase":    templatesBase,
		"DisplayTitle":     displayTitle,
		"Subtitle":         subtitle,
		"ListURL":          extensionsBase,
		"CancelURL":        extensionsBase,
		"HasConfiguration": hasConfig,
		"ConfigurationURL": configURL,
	})
	if errMsg != "" {
		data["Error"] = errMsg
	}
	h.render(w, r, displayTitle, "admin/extensions_form.html", data)
}

func (h *Handler) extensionTemplateOverride(w http.ResponseWriter, r *http.Request, idStr string) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	id, ok := parseID(idStr)
	if !ok {
		h.notFound(w, r)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	db, _ := sites.DB(r.Context())
	site, _ := sites.FromContext(r.Context())
	extMgr := h.chain.Extensions(site)
	var row models.Extension
	if err := db.First(&row, id).Error; err != nil {
		h.notFound(w, r)
		return
	}
	templatePath := formString(r, "template_path")
	source, err := extMgr.TemplateSource(row.Name, templatePath)
	if err != nil {
		h.renderExtensionForm(w, r, extMgr, row, err.Error())
		return
	}
	if source.OverridePath == "" || source.Content == "" {
		h.renderExtensionForm(w, r, extMgr, row, "Extension template source is unavailable.")
		return
	}
	target := filepath.Join(site.TemplateDir, filepath.FromSlash(source.OverridePath))
	if _, err := os.Stat(target); err == nil {
		h.renderExtensionForm(w, r, extMgr, row, "Template is already overridden.")
		return
	}
	if err := templatemgr.Save(site.TemplateDir, source.OverridePath, []byte(source.Content)); err != nil {
		h.renderExtensionForm(w, r, extMgr, row, err.Error())
		return
	}
	redirectList(w, r, templatesBase+"/edit?path="+url.QueryEscape(source.OverridePath))
}

func mergeExtensionMeta(live extensions.MetaSummary, row models.Extension) extensions.MetaSummary {
	if live.Available {
		return live
	}
	return extensions.MetaSummary{
		Available:   row.Title != "" || row.Description != "" || row.Version != "",
		Title:       row.Title,
		Description: row.Description,
		Version:     row.Version,
		Name:        row.Name,
	}
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}

func (h *Handler) extensionAction(w http.ResponseWriter, r *http.Request, idStr string, action func(context.Context, *extensions.Manager, models.Extension) error) {
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
	site, _ := sites.FromContext(r.Context())
	extMgr := h.chain.Extensions(site)
	var row models.Extension
	if err := db.First(&row, id).Error; err != nil {
		h.notFound(w, r)
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
		h.notFound(w, r)
		return
	}
	db, _ := sites.DB(r.Context())
	site, _ := sites.FromContext(r.Context())
	extMgr := h.chain.Extensions(site)
	var row models.Extension
	if err := db.First(&row, id).Error; err != nil {
		h.notFound(w, r)
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
		h.notFound(w, r)
		return
	}
	db, _ := sites.DB(r.Context())
	site, _ := sites.FromContext(r.Context())
	var row models.Extension
	if err := db.First(&row, id).Error; err != nil {
		h.notFound(w, r)
		return
	}
	h.chain.Extensions(site).Stop(row.Name)
	if err := db.Delete(&row).Error; err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	redirectList(w, r, extensionsBase)
}

func (h *Handler) extensionMoveSort(w http.ResponseWriter, r *http.Request, idStr string, direction int) {
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
	if err := extensionReorder(db, id, direction); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	redirectList(w, r, extensionsBase+listRedirectQuery(r))
}

func extensionReorder(db *gorm.DB, id uint, direction int) error {
	if direction == 0 {
		return nil
	}
	var all []models.Extension
	if err := db.Order("sort asc, extension_id asc").Find(&all).Error; err != nil {
		return err
	}
	idx := -1
	for i, row := range all {
		if row.ExtensionID == id {
			idx = i
			break
		}
	}
	if idx < 0 {
		return gorm.ErrRecordNotFound
	}
	target := idx + direction
	if target < 0 || target >= len(all) {
		return nil
	}
	all[idx], all[target] = all[target], all[idx]
	for i, row := range all {
		if row.Sort == i {
			continue
		}
		if err := db.Model(&models.Extension{}).Where("extension_id = ?", row.ExtensionID).Update("sort", i).Error; err != nil {
			return err
		}
	}
	return nil
}
