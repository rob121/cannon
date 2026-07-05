package admin

import (
	"context"
	"net/http"
	"net/url"
	"sort"
	"strings"

	"github.com/rob121/cannon/extension"
	"github.com/rob121/cannon/internal/blocks"
	"github.com/rob121/cannon/internal/content"
	"github.com/rob121/cannon/internal/extensions"
	"github.com/rob121/cannon/internal/models"
	"github.com/rob121/cannon/internal/router"
	"github.com/rob121/cannon/internal/settings"
	"github.com/rob121/cannon/internal/sites"
	"gorm.io/gorm"
)

const blocksBase = "/admin/blocks"

type blockListRow struct {
	models.Block
	TypeLabel   string
	CanMoveUp   bool
	CanMoveDown bool
}

type extensionBlockOption struct {
	Name   string
	Label  string
	Blocks []extension.BlockDefinition
}

func (h *Handler) blocks(w http.ResponseWriter, r *http.Request, path string) {
	parts := pathParts("/blocks", path)
	switch {
	case len(parts) == 0:
		h.blockList(w, r)
	case parts[0] == "new":
		h.blockForm(w, r, 0)
	case len(parts) == 2 && parts[1] == "delete":
		h.blockDelete(w, r, parts[0])
	case len(parts) == 2 && parts[1] == "toggle-status":
		h.blockToggleStatus(w, r, parts[0])
	case len(parts) == 2 && parts[1] == "move-up":
		h.blockMoveSort(w, r, parts[0], -1)
	case len(parts) == 2 && parts[1] == "move-down":
		h.blockMoveSort(w, r, parts[0], 1)
	default:
		id, ok := parseID(parts[0])
		if !ok {
			h.notFound(w, r)
			return
		}
		h.blockForm(w, r, id)
	}
}

func (h *Handler) blockList(w http.ResponseWriter, r *http.Request) {
	db, _ := sites.DB(r.Context())
	page := parsePage(r)
	spaceFilter := r.URL.Query().Get("space")

	q := db.Model(&models.Block{})
	if spaceFilter != "" {
		q = q.Where("space = ?", spaceFilter)
	}
	var total int64
	q.Count(&total)

	data := listPage(r, page, total, blocksBase,
		"Assign HTML, Markdown, or extension blocks to template spaces.",
		"Add Block", map[string]any{"ActiveNav": "blocks"})
	order := applyListSort(r, data, map[string]string{
		"name": "name", "space": "space", "type": "type", "sort": "sort", "status": "status",
	}, "sort")
	sortParam := strings.TrimSpace(r.URL.Query().Get("sort"))
	if sortParam == "" || sortParam == "sort" {
		order = "sort asc, block_id asc"
	}

	var rows []models.Block
	listQ := db.Model(&models.Block{})
	if spaceFilter != "" {
		listQ = listQ.Where("space = ?", spaceFilter)
	}
	listQ.Offset((page - 1) * pageSizeFor(r)).Limit(pageSizeFor(r)).Order(order).Find(&rows)

	position := blockSortPositions(db, spaceFilter)

	listRows := make([]blockListRow, 0, len(rows))
	for _, row := range rows {
		item := blockListRow{Block: row, TypeLabel: blockTypeLabel(row)}
		if pos, ok := position[row.BlockID]; ok {
			item.CanMoveUp = pos.canMoveUp
			item.CanMoveDown = pos.canMoveDown
		}
		listRows = append(listRows, item)
	}

	spaces, _ := blocks.DistinctSpaces(db)
	data["Rows"] = listRows
	data["SpaceFilter"] = spaceFilter
	data["Spaces"] = spaces
	data["ListQuery"] = listQueryFromData(data)
	if spaceFilter != "" {
		data["PageActionURL"] = blocksBase + "/new?space=" + url.QueryEscape(spaceFilter)
	}
	h.render(w, r, "Blocks", "admin/blocks.html", data)
}

func (h *Handler) blockForm(w http.ResponseWriter, r *http.Request, id uint) {
	db, _ := sites.DB(r.Context())
	site, _ := sites.FromContext(r.Context())
	extMgr := h.chain.Extensions(site)
	_ = extMgr.Bootstrap(r.Context())

	isNew := id == 0
	var row models.Block
	if !isNew {
		if err := db.Preload("Groups").First(&row, id).Error; err != nil {
			h.notFound(w, r)
			return
		}
	} else {
		row.Type = models.BlockTypeHTML
		row.Status = models.StatusActive
		if preset := r.URL.Query().Get("space"); preset != "" {
			row.Space = preset
		}
	}

	meta, _ := blocks.ParseMetadata(row.Metadata)
	extOptions := extensionBlockOptions(extMgr, db)
	allGroups := loadFrontendGroups(db)

	if r.Method == http.MethodPost {
		if err := r.ParseForm(); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		row.Name = formString(r, "name")
		row.Space = formString(r, "space")
		row.Type = models.BlockType(formString(r, "type"))
		row.Status = formStatus(r)
		row.Sort = formInt(r, "sort", 0)
		row.ExtensionName = formString(r, "extension_name")
		row.ExtensionBlockID = formString(r, "extension_block_id")
		spaceFilter := formString(r, "space_filter")
		if row.Type == models.BlockTypeExtension {
			if row.ExtensionName == "" || row.ExtensionBlockID == "" {
				metaRaw, _ := blocks.MetadataFromFormValues(string(row.Type), r.FormValue("content"), r.Form)
				meta, _ = blocks.ParseMetadata(metaRaw)
				row.Metadata = metaRaw
				h.renderBlockForm(w, r, row, meta, extOptions, allGroups, isNew, spaceFilter, "Extension and block are required.")
				return
			}
		}
		metaRaw, err := blocks.MetadataFromFormValues(string(row.Type), r.FormValue("content"), r.Form)
		if err != nil {
			metaRaw, _ = blocks.MetadataFromFormValues(string(row.Type), r.FormValue("content"), r.Form)
			meta, _ = blocks.ParseMetadata(metaRaw)
			row.Metadata = metaRaw
			h.renderBlockForm(w, r, row, meta, extOptions, allGroups, isNew, spaceFilter, err.Error())
			return
		}
		row.Metadata = metaRaw

		var saveErr error
		if isNew {
			saveErr = db.Create(&row).Error
		} else {
			saveErr = db.Save(&row).Error
		}
		if saveErr != nil {
			metaRaw, _ := blocks.MetadataFromFormValues(string(row.Type), r.FormValue("content"), r.Form)
			meta, _ = blocks.ParseMetadata(metaRaw)
			row.Metadata = metaRaw
			h.renderBlockForm(w, r, row, meta, extOptions, allGroups, isNew, spaceFilter, saveErr.Error())
			return
		}
		if err := replaceFormGroups(db, &row, r); err != nil {
			h.renderBlockForm(w, r, row, meta, extOptions, allGroups, isNew, spaceFilter, err.Error())
			return
		}
		invalidateBlocksDataCache(r.Context())
		redirectList(w, r, blocksBase+listRedirectQuery(r))
		return
	}

	title := "Add Block"
	if !isNew {
		title = "Edit Block"
	}
	spaceFilter := r.URL.Query().Get("space")
	h.render(w, r, title, "admin/blocks_form.html", blockFormData(r.Context(), db, row, meta, extOptions, allGroups, isNew, spaceFilter, ""))
}

func (h *Handler) blockDelete(w http.ResponseWriter, r *http.Request, idStr string) {
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
	if err := db.Exec("DELETE FROM block_groups WHERE block_block_id = ?", id).Error; err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if err := db.Delete(&models.Block{}, id).Error; err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	invalidateBlocksDataCache(r.Context())
	redirectList(w, r, blocksBase+listRedirectQuery(r))
}

func (h *Handler) renderBlockForm(w http.ResponseWriter, r *http.Request, row models.Block, meta blocks.Metadata, extOptions []extensionBlockOption, allGroups []models.Group, isNew bool, spaceFilter, errMsg string) {
	title := "Add Block"
	if !isNew {
		title = "Edit Block"
	}
	db, _ := sites.DB(r.Context())
	data := blockFormData(r.Context(), db, row, meta, extOptions, allGroups, isNew, spaceFilter, errMsg)
	h.render(w, r, title, "admin/blocks_form.html", data)
}

func blockFormData(ctx context.Context, db *gorm.DB, row models.Block, meta blocks.Metadata, extOptions []extensionBlockOption, allGroups []models.Group, isNew bool, spaceFilter, errMsg string) map[string]any {
	categories, _ := content.CategoryTreeAll(ctx)
	var tags []models.Tag
	db.Order("name asc").Find(&tags)
	var allMenus []models.Menu
	db.Order("menu_name asc").Find(&allMenus)
	routeMode := meta.RouteMode
	if routeMode == "" {
		routeMode = blocks.RouteModeAll
	}
	_ = router.EnsureDefaultRoute(db)
	var allRoutes []models.Route
	db.Order("is_default desc, name asc, path asc").Find(&allRoutes)
	routeSections := buildBlockRouteSections(allRoutes)
	selectedRouteIDs := meta.RouteIDs
	templateDir := ""
	if site, err := sites.FromContext(ctx); err == nil && site != nil {
		templateDir = site.TemplateDir
	}
	frontendTheme, _ := settings.FrontendTheme(ctx)
	wrapperTemplates, _ := blocks.ListWrapperTemplates(templateDir, frontendTheme)
	if meta.TemplateWrapper != "" && !wrapperOptionContains(wrapperTemplates, meta.TemplateWrapper) {
		wrapperTemplates = append(wrapperTemplates, blocks.WrapperOption{
			Path:  meta.TemplateWrapper,
			Label: meta.TemplateWrapper + " (custom)",
		})
		sortWrapperOptions(wrapperTemplates)
	}
	data := formData(map[string]any{
		"ActiveNav":        "blocks",
		"Row":              row,
		"Meta":             meta,
		"RouteMode":        routeMode,
		"SelectedRouteIDs": selectedRouteIDs,
		"RouteSections":    routeSections,
		"RouteTotal":       len(allRoutes),
		"RouteSelected":    len(selectedRouteIDs),
		"ExtensionOptions": extOptions,
		"BlockData":        blocks.MetadataStringMap(row.Metadata),
		"IsNew":            isNew,
		"BasePath":         blocksBase,
		"SpaceFilter":      spaceFilter,
		"CancelURL":        blocksBase + listQueryFromData(map[string]any{"SpaceFilter": spaceFilter}),
		"AllGroups":        allGroups,
		"SelectedIDs":      defaultGroupSelectedIDs(db, row.Groups, isNew),
		"Categories":       content.FlattenCategoryOptions(categories),
		"Tags":             tags,
		"AllMenus":         allMenus,
		"WrapperTemplates": wrapperTemplates,
	})
	if errMsg != "" {
		data["Error"] = errMsg
	}
	return data
}

func extensionBlockOptions(extMgr *extensions.Manager, db *gorm.DB) []extensionBlockOption {
	menuNames := extensionMenuNames(db)
	out := make([]extensionBlockOption, 0)
	for _, rt := range extMgr.BlockRuntimes() {
		if len(rt.Blocks) == 0 {
			continue
		}
		out = append(out, extensionBlockOption{
			Name:   rt.Model.Name,
			Label:  extensionMenuLabel(rt.Model.Name, menuNames),
			Blocks: rt.Blocks,
		})
	}
	return out
}

func blockTypeLabel(row models.Block) string {
	switch row.Type {
	case models.BlockTypeHTML:
		return "HTML"
	case models.BlockTypeMarkdown:
		return "Markdown"
	case models.BlockTypeExtension:
		label := row.ExtensionName
		if row.ExtensionBlockID != "" {
			label += " / " + row.ExtensionBlockID
		}
		return "Extension (" + label + ")"
	case models.BlockTypeContent:
		meta, _ := blocks.ParseMetadata(row.Metadata)
		mode := meta.ContentMode
		if mode == "" {
			mode = "latest"
		}
		return "Content (" + mode + ")"
	case models.BlockTypeLogin:
		return "Login"
	case models.BlockTypeMenuVertical:
		meta, _ := blocks.ParseMetadata(row.Metadata)
		if meta.MenuName != "" {
			return "Menu Vertical (" + meta.MenuName + ")"
		}
		return "Menu Vertical"
	case models.BlockTypeMenuHorizontal:
		meta, _ := blocks.ParseMetadata(row.Metadata)
		if meta.MenuName != "" {
			return "Menu Horizontal (" + meta.MenuName + ")"
		}
		return "Menu Horizontal"
	case models.BlockTypeSearchHorizontal:
		return "Search Horizontal"
	case models.BlockTypeSearchVertical:
		return "Search Vertical"
	default:
		return string(row.Type)
	}
}

type blockSortPosition struct {
	canMoveUp   bool
	canMoveDown bool
}

func blockSortPositions(db *gorm.DB, spaceFilter string) map[uint]blockSortPosition {
	var all []models.Block
	q := db.Order("space asc, sort asc, block_id asc")
	if spaceFilter != "" {
		q = q.Where("space = ?", spaceFilter)
	}
	if err := q.Find(&all).Error; err != nil {
		return map[uint]blockSortPosition{}
	}
	bySpace := make(map[string][]models.Block)
	for _, row := range all {
		bySpace[row.Space] = append(bySpace[row.Space], row)
	}
	out := make(map[uint]blockSortPosition, len(all))
	for _, rows := range bySpace {
		last := len(rows) - 1
		for i, row := range rows {
			out[row.BlockID] = blockSortPosition{
				canMoveUp:   i > 0,
				canMoveDown: i < last,
			}
		}
	}
	return out
}

func (h *Handler) blockMoveSort(w http.ResponseWriter, r *http.Request, idStr string, direction int) {
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
	if err := blockReorder(db, id, direction); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	invalidateBlocksDataCache(r.Context())
	redirectList(w, r, blocksBase+listRedirectQuery(r))
}

func blockReorder(db *gorm.DB, id uint, direction int) error {
	if direction == 0 {
		return nil
	}
	var row models.Block
	if err := db.First(&row, id).Error; err != nil {
		return err
	}
	var all []models.Block
	if err := db.Where("space = ?", row.Space).Order("sort asc, block_id asc").Find(&all).Error; err != nil {
		return err
	}
	idx := -1
	for i, item := range all {
		if item.BlockID == id {
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
	for i, item := range all {
		if item.Sort == i {
			continue
		}
		if err := db.Model(&models.Block{}).Where("block_id = ?", item.BlockID).Update("sort", i).Error; err != nil {
			return err
		}
	}
	return nil
}

func wrapperOptionContains(options []blocks.WrapperOption, path string) bool {
	for _, opt := range options {
		if opt.Path == path {
			return true
		}
	}
	return false
}

func sortWrapperOptions(options []blocks.WrapperOption) {
	sort.Slice(options, func(i, j int) bool { return options[i].Path < options[j].Path })
}