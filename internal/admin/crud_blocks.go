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
	TypeLabel     string
	SpacesDisplay string
	CanMoveUp     bool
	CanMoveDown   bool
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
	_ = blocks.EnsureAdminDashboardBlocks(r.Context())
	db, _ := sites.DB(r.Context())
	page := parsePage(r)
	spaceFilter := strings.TrimSpace(r.URL.Query().Get("space"))

	q := db.Model(&models.Block{})
	if spaceFilter != "" {
		q = q.Joins("JOIN block_spaces ON block_spaces.block_id = blocks.block_id").
			Where("block_spaces.space = ?", spaceFilter)
	}
	var total int64
	q.Count(&total)

	data := listPage(r, page, total, blocksBase,
		"Assign HTML, Markdown, or extension blocks to template spaces.",
		"Add Block", map[string]any{"ActiveNav": "blocks"})
	order := applyListSort(r, data, map[string]string{
		"name": "blocks.name", "space": "blocks.space", "type": "blocks.type", "sort": "blocks.sort", "status": "blocks.status",
	}, "sort")
	sortParam := strings.TrimSpace(r.URL.Query().Get("sort"))
	dirParam := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("dir")))
	defaultSort := sortParam == "" || sortParam == "sort"
	if defaultSort {
		if spaceFilter != "" {
			if dirParam == "desc" {
				order = "block_spaces.sort DESC, blocks.block_id DESC"
			} else {
				order = "block_spaces.sort ASC, blocks.block_id ASC"
			}
		} else {
			// Prefer denormalized sort; refine in-memory by primary space for stable row order.
			if dirParam == "desc" {
				order = "blocks.sort DESC, blocks.name ASC, blocks.block_id ASC"
			} else {
				order = "blocks.sort ASC, blocks.name ASC, blocks.block_id ASC"
			}
		}
		data["Sort"] = "sort"
		if dirParam != "desc" {
			data["Dir"] = "asc"
		}
	}

	var rows []models.Block
	listQ := db.Model(&models.Block{}).Preload("Spaces")
	pageSize := pageSizeFor(r)
	if spaceFilter != "" {
		listQ.Joins("JOIN block_spaces ON block_spaces.block_id = blocks.block_id").
			Where("block_spaces.space = ?", spaceFilter).
			Select("blocks.*, block_spaces.sort AS sort").
			Offset((page - 1) * pageSize).Limit(pageSize).Order(order).Find(&rows)
	} else if defaultSort {
		// Sort by primary space + membership sort across the full set, then page.
		listQ.Order(order).Find(&rows)
		sortBlocksByPrimarySpace(rows, dirParam == "desc")
		for i := range rows {
			rows[i].Sort = primarySpaceSort(rows[i])
		}
		start := (page - 1) * pageSize
		if start >= len(rows) {
			rows = nil
		} else {
			end := start + pageSize
			if end > len(rows) {
				end = len(rows)
			}
			rows = rows[start:end]
		}
	} else {
		listQ.Offset((page - 1) * pageSize).Limit(pageSize).Order(order).Find(&rows)
		for i := range rows {
			rows[i].Sort = primarySpaceSort(rows[i])
		}
	}

	var position map[uint]blockSortPosition
	if spaceFilter != "" {
		position = blockSortPositions(db, spaceFilter)
	} else {
		position = blockSortPositionsByPrimarySpace(rows)
	}

	listRows := make([]blockListRow, 0, len(rows))
	for _, row := range rows {
		item := blockListRow{
			Block:         row,
			TypeLabel:     blockTypeLabel(row),
			SpacesDisplay: strings.Join(blocks.SpaceNames(row), ", "),
		}
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
	data["ReorderEnabled"] = true
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
		if err := db.Preload("Groups").Preload("Spaces").First(&row, id).Error; err != nil {
			h.notFound(w, r)
			return
		}
	} else {
		row.Type = models.BlockTypeHTML
		row.Status = models.StatusActive
		if preset := strings.TrimSpace(r.URL.Query().Get("space")); preset != "" {
			row.Spaces = []models.BlockSpace{{Space: preset}}
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
		row.Type = models.BlockType(formString(r, "type"))
		row.Status = formStatus(r)
		row.ExtensionName = formString(r, "extension_name")
		row.ExtensionBlockID = formString(r, "extension_block_id")
		spaceFilter := formString(r, "space_filter")
		spaceValues := blocks.NormalizeSpaces(r.Form["spaces"])
		if len(spaceValues) == 0 {
			// Accept legacy single "space" field if present.
			spaceValues = blocks.NormalizeSpaces([]string{formString(r, "space")})
		}
		row.Spaces = make([]models.BlockSpace, 0, len(spaceValues))
		for _, space := range spaceValues {
			row.Spaces = append(row.Spaces, models.BlockSpace{Space: space})
		}
		if len(spaceValues) > 0 {
			row.Space = spaceValues[0]
		} else {
			row.Space = ""
		}
		assignFormSpaces := func() {
			row.Spaces = make([]models.BlockSpace, 0, len(spaceValues))
			for _, space := range spaceValues {
				row.Spaces = append(row.Spaces, models.BlockSpace{Space: space})
			}
		}
		if row.Type == models.BlockTypeExtension {
			if row.ExtensionName == "" || row.ExtensionBlockID == "" {
				metaRaw, _ := blocks.MetadataFromFormValues(string(row.Type), r.FormValue("content"), r.Form)
				meta, _ = blocks.ParseMetadata(metaRaw)
				row.Metadata = metaRaw
				assignFormSpaces()
				h.renderBlockForm(w, r, row, meta, extOptions, allGroups, isNew, spaceFilter, "Extension and block are required.")
				return
			}
		}
		if len(spaceValues) == 0 {
			metaRaw, _ := blocks.MetadataFromFormValues(string(row.Type), r.FormValue("content"), r.Form)
			meta, _ = blocks.ParseMetadata(metaRaw)
			row.Metadata = metaRaw
			h.renderBlockForm(w, r, row, meta, extOptions, allGroups, isNew, spaceFilter, blocks.ErrNoSpaces.Error())
			return
		}
		metaRaw, err := blocks.MetadataFromFormValues(string(row.Type), r.FormValue("content"), r.Form)
		if err != nil {
			metaRaw, _ = blocks.MetadataFromFormValues(string(row.Type), r.FormValue("content"), r.Form)
			meta, _ = blocks.ParseMetadata(metaRaw)
			row.Metadata = metaRaw
			assignFormSpaces()
			h.renderBlockForm(w, r, row, meta, extOptions, allGroups, isNew, spaceFilter, err.Error())
			return
		}
		row.Metadata = metaRaw

		// Persist memberships via ReplaceBlockSpaces; avoid GORM association create.
		savedSpaces := row.Spaces
		row.Spaces = nil
		var saveErr error
		if isNew {
			saveErr = db.Omit("Spaces").Create(&row).Error
		} else {
			saveErr = db.Omit("Spaces").Save(&row).Error
		}
		row.Spaces = savedSpaces
		if saveErr != nil {
			metaRaw, _ := blocks.MetadataFromFormValues(string(row.Type), r.FormValue("content"), r.Form)
			meta, _ = blocks.ParseMetadata(metaRaw)
			row.Metadata = metaRaw
			h.renderBlockForm(w, r, row, meta, extOptions, allGroups, isNew, spaceFilter, saveErr.Error())
			return
		}
		if err := blocks.ReplaceBlockSpaces(db, row.BlockID, spaceValues); err != nil {
			h.renderBlockForm(w, r, row, meta, extOptions, allGroups, isNew, spaceFilter, err.Error())
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
	if err := db.Where("block_id = ?", id).Delete(&models.BlockSpace{}).Error; err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
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
	adminTheme, _ := settings.AdminTheme(ctx)
	wrapperTemplates, _ := blocks.ListWrapperTemplates(templateDir, frontendTheme)
	if meta.TemplateWrapper != "" && !wrapperOptionContains(wrapperTemplates, meta.TemplateWrapper) {
		wrapperTemplates = append(wrapperTemplates, blocks.WrapperOption{
			Path:  meta.TemplateWrapper,
			Label: meta.TemplateWrapper + " (custom)",
		})
		sortWrapperOptions(wrapperTemplates)
	}
	spaceNames := blocks.SpaceNames(row)
	dbSpaces, _ := blocks.DistinctSpaces(db)
	knownSpaces, _ := blocks.ListKnownSpaces(templateDir, frontendTheme, adminTheme, dbSpaces)
	extSpaces := extensionBlockSpaces(extOptions, row.ExtensionName, row.ExtensionBlockID)
	suggestedSpaces := knownSpaces
	if len(extSpaces) > 0 {
		suggestedSpaces = extSpaces
	}
	data := formData(map[string]any{
		"ActiveNav":        "blocks",
		"Row":              row,
		"SpaceNames":       spaceNames,
		"KnownSpaces":      knownSpaces,
		"ExtensionSpaces":  extSpaces,
		"SuggestedSpaces":  suggestedSpaces,
		"Meta":             meta,
		"RouteMode":        routeMode,
		"SelectedRouteIDs": selectedRouteIDs,
		"RouteSections":    routeSections,
		"RouteTotal":       len(allRoutes),
		"RouteSelected":    len(selectedRouteIDs),
		"ExtensionOptions": extOptions,
		"BlockData":        blocks.MetadataStringMap(row.Metadata),
		"BlockDataRaw":     blockDataRawMap(row.Metadata),
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

func extensionBlockSpaces(options []extensionBlockOption, extensionName, blockID string) []string {
	extensionName = strings.TrimSpace(extensionName)
	blockID = strings.TrimSpace(blockID)
	if extensionName == "" || blockID == "" {
		return nil
	}
	for _, opt := range options {
		if opt.Name != extensionName {
			continue
		}
		for _, def := range opt.Blocks {
			if def.ID != blockID {
				continue
			}
			return blocks.NormalizeSpaces(def.Spaces)
		}
	}
	return nil
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
		return string(row.Type)
	}
}

type blockSortPosition struct {
	canMoveUp   bool
	canMoveDown bool
}

func blockSortPositions(db *gorm.DB, spaceFilter string) map[uint]blockSortPosition {
	if strings.TrimSpace(spaceFilter) == "" {
		return map[uint]blockSortPosition{}
	}
	var memberships []models.BlockSpace
	if err := db.Where("space = ?", spaceFilter).Order("sort ASC, block_id ASC").Find(&memberships).Error; err != nil {
		return map[uint]blockSortPosition{}
	}
	out := make(map[uint]blockSortPosition, len(memberships))
	last := len(memberships) - 1
	for i, row := range memberships {
		out[row.BlockID] = blockSortPosition{
			canMoveUp:   i > 0,
			canMoveDown: i < last,
		}
	}
	return out
}

// blockSortPositionsByPrimarySpace enables up/down within each primary-space group
// on the unfiltered blocks list.
func blockSortPositionsByPrimarySpace(rows []models.Block) map[uint]blockSortPosition {
	grouped := map[string][]uint{}
	order := make([]string, 0)
	for _, row := range rows {
		space := blocks.PrimarySpace(row)
		if _, seen := grouped[space]; !seen {
			order = append(order, space)
		}
		grouped[space] = append(grouped[space], row.BlockID)
	}
	out := make(map[uint]blockSortPosition, len(rows))
	for _, space := range order {
		ids := grouped[space]
		last := len(ids) - 1
		for i, id := range ids {
			out[id] = blockSortPosition{
				canMoveUp:   i > 0,
				canMoveDown: i < last,
			}
		}
	}
	return out
}

func sortBlocksByPrimarySpace(rows []models.Block, desc bool) {
	sort.SliceStable(rows, func(i, j int) bool {
		si, sj := blocks.PrimarySpace(rows[i]), blocks.PrimarySpace(rows[j])
		if si != sj {
			if desc {
				return si > sj
			}
			return si < sj
		}
		oi, oj := primarySpaceSort(rows[i]), primarySpaceSort(rows[j])
		if oi != oj {
			if desc {
				return oi > oj
			}
			return oi < oj
		}
		return rows[i].BlockID < rows[j].BlockID
	})
}

func primarySpaceSort(row models.Block) int {
	space := blocks.PrimarySpace(row)
	for _, membership := range row.Spaces {
		if membership.Space == space {
			return membership.Sort
		}
	}
	return row.Sort
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
	spaceFilter := strings.TrimSpace(r.URL.Query().Get("space"))
	if spaceFilter == "" {
		var row models.Block
		if err := db.Preload("Spaces").First(&row, id).Error; err != nil {
			h.notFound(w, r)
			return
		}
		spaceFilter = blocks.PrimarySpace(row)
		if spaceFilter == "" {
			http.Error(w, "block has no space to reorder", http.StatusBadRequest)
			return
		}
	}
	if err := blockReorder(db, id, spaceFilter, direction); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	invalidateBlocksDataCache(r.Context())
	redirectList(w, r, blocksBase+listRedirectQuery(r))
}

func blockReorder(db *gorm.DB, id uint, space string, direction int) error {
	if direction == 0 {
		return nil
	}
	space = strings.TrimSpace(space)
	if space == "" {
		return blocks.ErrNoSpaces
	}
	var all []models.BlockSpace
	if err := db.Where("space = ?", space).Order("sort ASC, block_id ASC").Find(&all).Error; err != nil {
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
		if item.Sort != i {
			if err := db.Model(&models.BlockSpace{}).Where("block_space_id = ?", item.BlockSpaceID).Update("sort", i).Error; err != nil {
				return err
			}
		}
		// Keep denormalized blocks.sort in sync for unfiltered list ordering.
		var row models.Block
		if err := db.Preload("Spaces").First(&row, item.BlockID).Error; err != nil {
			return err
		}
		if blocks.PrimarySpace(row) == space && row.Sort != i {
			if err := db.Model(&models.Block{}).Where("block_id = ?", item.BlockID).Update("sort", i).Error; err != nil {
				return err
			}
		}
	}
	return nil
}

func blockDataRawMap(raw string) map[string]any {
	out, err := blocks.MetadataMap(raw)
	if err != nil || out == nil {
		return map[string]any{}
	}
	return out
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
