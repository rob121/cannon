package admin

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/rob121/cannon/extension"
	"github.com/rob121/cannon/internal/content"
	"github.com/rob121/cannon/internal/controllers"
	_ "github.com/rob121/cannon/internal/controllers/auth"
	_ "github.com/rob121/cannon/internal/controllers/content"
	"github.com/rob121/cannon/internal/extensions"
	"github.com/rob121/cannon/internal/models"
	"github.com/rob121/cannon/internal/routemeta"
	"github.com/rob121/cannon/internal/router"
	"github.com/rob121/cannon/internal/settings"
	"github.com/rob121/cannon/internal/sites"
	"github.com/rob121/cannon/internal/templatemgr"
	"gorm.io/gorm"
)

const routesBase = "/admin/routes"

type extensionRouteOption struct {
	Name      string
	Label     string
	Pages     []extension.PageDefinition
	Endpoints []extension.EndpointDefinition
}

type controllerRouteOption struct {
	ID     string
	Title  string
	Actions []controllers.ActionDefinition
}

func controllerOptions(selectedID string) []controllerRouteOption {
	out := make([]controllerRouteOption, 0, len(controllers.Definitions()))
	for _, def := range controllers.Definitions() {
		out = append(out, controllerRouteOption{
			ID:      def.ID,
			Title:   def.Title,
			Actions: def.Actions,
		})
	}
	if selectedID != "" {
		found := false
		for _, opt := range out {
			if opt.ID == selectedID {
				found = true
				break
			}
		}
		if !found {
			out = append(out, controllerRouteOption{ID: selectedID, Title: selectedID})
		}
	}
	return out
}

func (h *Handler) routes(w http.ResponseWriter, r *http.Request, path string) {
	parts := pathParts("/routes", path)
	switch {
	case len(parts) == 0:
		h.routeList(w, r)
	case parts[0] == "new":
		h.routeForm(w, r, 0)
	case len(parts) == 2 && parts[1] == "delete":
		h.routeDelete(w, r, parts[0])
	case len(parts) == 2 && parts[1] == "toggle-status":
		h.routeToggleStatus(w, r, parts[0])
	default:
		id, ok := parseID(parts[0])
		if !ok {
			h.notFound(w, r)
			return
		}
		h.routeForm(w, r, id)
	}
}

func (h *Handler) routeList(w http.ResponseWriter, r *http.Request) {
	db, _ := sites.DB(r.Context())
	page := parsePage(r)
	var rows []models.Route
	var total int64
	db.Model(&models.Route{}).Count(&total)
	data := listPage(r, page, total, routesBase,
		"Custom URL paths and handlers. Built-in content, auth, and account routes are listed separately below.",
		"Add Route", map[string]any{"ActiveNav": "routes"})
	order := applyListSort(r, data, map[string]string{
		"name": "name", "path": "path", "type": "type", "status": "status",
	}, "name")
	db.Offset((page - 1) * pageSizeFor(r)).Limit(pageSizeFor(r)).Order(order).Find(&rows)
	userRows := make([]models.Route, 0, len(rows))
	for _, row := range rows {
		if !router.IsBuiltinControllerRoute(row) {
			userRows = append(userRows, row)
		}
	}
	data["Rows"] = userRows
	data["SystemRoutes"] = router.SystemRoutes()
	data["BuiltinRoutes"] = router.BuiltinControllerRoutes()
	data["BuiltinRouteRows"] = loadBuiltinRouteRows(db)
	h.render(w, r, "Routes", "admin/routes.html", data)
}

type builtinRouteRow struct {
	RouteID   uint
	Name      string
	Path      string
	Prefix    string
	Controller string
	ControllerAction string
	Methods   string
	ShowTitle bool
}

func loadBuiltinRouteRows(db *gorm.DB) []builtinRouteRow {
	out := make([]builtinRouteRow, 0, len(router.BuiltinControllerRoutes()))
	for _, br := range router.BuiltinControllerRoutes() {
		row := builtinRouteRow{
			Name:             br.Name,
			Path:             br.Path,
			Prefix:           br.Prefix,
			Controller:       br.Controller,
			ControllerAction: br.ControllerAction,
			Methods:          br.Methods,
			ShowTitle:        true,
		}
		var dbRow models.Route
		if err := db.Where("path = ?", br.Path).First(&dbRow).Error; err == nil {
			row.RouteID = dbRow.RouteID
			row.ShowTitle = dbRow.ShowTitle
		}
		out = append(out, row)
	}
	return out
}

func (h *Handler) routeForm(w http.ResponseWriter, r *http.Request, id uint) {
	db, _ := sites.DB(r.Context())
	site, _ := sites.FromContext(r.Context())
	extMgr := h.chain.Extensions(site)
	_ = extMgr.Bootstrap(r.Context())
	extMgr.EnsurePageDefinitions(r.Context())
	extMgr.EnsureEndpointDefinitions(r.Context())

	isNew := id == 0
	var row models.Route
	if isNew {
		row.ShowTitle = true
	}
	if !isNew {
		if err := db.Preload("Groups").First(&row, id).Error; err != nil {
			h.notFound(w, r)
			return
		}
	}
	allGroups := loadActiveGroups(db)
	extOptions := extensionRouteOptions(extMgr, db, row.ExtensionName)

	if r.Method == http.MethodPost {
		if err := r.ParseForm(); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		row.Name = formString(r, "name")
		row.Path = formString(r, "path")
		row.Type = models.RouteType(formString(r, "type"))
		row.Status = formStatus(r)
		row.Target = formString(r, "target")
		row.ExtensionName = formString(r, "extension_name")
		row.ExtensionPageID = formString(r, "extension_page_id")
		row.ExtensionEndpointID = formString(r, "extension_endpoint_id")
		row.Controller = formString(r, "controller")
		row.ControllerAction = formString(r, "controller_action")
		row.IsDefault = formBool(r, "is_default")
		row.ShowTitle = formBool(r, "show_title")

		if row.Type == models.RouteTypeExtension {
			if row.ExtensionName == "" || row.ExtensionPageID == "" {
				metaRaw, _ := routemeta.MetadataFromForm(r.Form)
				row.Metadata = metaRaw
				h.renderRouteForm(w, r, row, allGroups, extOptions, controllerOptions(row.Controller), isNew, "Extension and page are required.")
				return
			}
		}
		if row.Type == models.RouteTypeExtensionEndpoint {
			if row.ExtensionName == "" || row.ExtensionEndpointID == "" {
				metaRaw, _ := routemeta.MetadataFromForm(r.Form)
				row.Metadata = metaRaw
				h.renderRouteForm(w, r, row, allGroups, extOptions, controllerOptions(row.Controller), isNew, "Extension and endpoint are required.")
				return
			}
		}
		if row.Type == models.RouteTypeController {
			if row.Controller == "" || row.ControllerAction == "" {
				metaRaw, _ := routemeta.MetadataFromForm(r.Form)
				row.Metadata = metaRaw
				h.renderRouteForm(w, r, row, allGroups, extOptions, controllerOptions(row.Controller), isNew, "Controller and action are required.")
				return
			}
		}
		metaRaw, err := routemeta.MetadataFromForm(r.Form)
		if err != nil {
			h.renderRouteForm(w, r, row, allGroups, extOptions, controllerOptions(row.Controller), isNew, err.Error())
			return
		}
		metaRaw, err = routemeta.SetMetadataString(metaRaw, "template", formString(r, "template_override"))
		if err != nil {
			h.renderRouteForm(w, r, row, allGroups, extOptions, controllerOptions(row.Controller), isNew, err.Error())
			return
		}
		row.Metadata = metaRaw
		if row.Type == models.RouteTypeController {
			if err := validateControllerRoute(row); err != nil {
				h.renderRouteForm(w, r, row, allGroups, extOptions, controllerOptions(row.Controller), isNew, err.Error())
				return
			}
		}
		if router.ConflictsWithReservedPath(row.Path) {
			if isNew || !router.IsBuiltinControllerRoute(row) {
				h.renderRouteForm(w, r, row, allGroups, extOptions, controllerOptions(row.Controller), isNew, "Path conflicts with a reserved system or built-in controller route.")
				return
			}
		}

		if row.Type != models.RouteTypeExtension && row.Type != models.RouteTypeExtensionEndpoint {
			row.ExtensionName = ""
			row.ExtensionPageID = ""
			row.ExtensionEndpointID = ""
			if row.Type != models.RouteTypeController {
				row.Metadata = ""
			}
		}
		if row.Type == models.RouteTypeExtension {
			row.ExtensionEndpointID = ""
		}
		if row.Type == models.RouteTypeExtensionEndpoint {
			row.ExtensionPageID = ""
		}
		if row.Type != models.RouteTypeController {
			row.Controller = ""
			row.ControllerAction = ""
		}
		if row.Type != models.RouteTypeURL && row.Type != models.RouteTypeLocalFile {
			row.Target = ""
		}

		var saveErr error
		saveErr = db.Transaction(func(tx *gorm.DB) error {
			if isNew {
				if err := tx.Create(&row).Error; err != nil {
					return err
				}
			} else if err := tx.Save(&row).Error; err != nil {
				return err
			}
			if row.IsDefault {
				return tx.Model(&models.Route{}).Where("route_id <> ?", row.RouteID).Update("is_default", false).Error
			}
			return nil
		})
		if saveErr != nil {
			h.renderRouteForm(w, r, row, allGroups, extOptions, controllerOptions(row.Controller), isNew, saveErr.Error())
			return
		}
		if err := replaceFormGroups(db, &row, r); err != nil {
			h.renderRouteForm(w, r, row, allGroups, extOptions, controllerOptions(row.Controller), isNew, err.Error())
			return
		}
		redirectList(w, r, routesBase)
		return
	}
	h.renderRouteForm(w, r, row, allGroups, extOptions, controllerOptions(row.Controller), isNew, "")
}

func (h *Handler) renderRouteForm(w http.ResponseWriter, r *http.Request, row models.Route, allGroups []models.Group, extOptions []extensionRouteOption, ctrlOptions []controllerRouteOption, isNew bool, errMsg string) {
	title := "Add Route"
	if !isNew {
		title = "Edit Route"
	}
	db, _ := sites.DB(r.Context())
	categories, _ := content.CategoryTree(r.Context())
	tags, _ := content.ListTags(r.Context())
	var items []models.Item
	db.Where("status <> ?", models.ItemStatusTrashed).Order("title asc").Limit(250).Find(&items)
	var users []models.User
	db.Where("status = ?", models.StatusActive).Order("username asc").Limit(250).Find(&users)
	site, _ := sites.FromContext(r.Context())
	templateDir := ""
	frontendTheme := ""
	if site != nil {
		templateDir = site.TemplateDir
	}
	if theme, err := settings.FrontendTheme(r.Context()); err == nil {
		frontendTheme = theme
	}
	controllerTemplateOptions, _ := allControllerTemplateOptions(templateDir, frontendTheme)
	templateOverride := routemeta.MetadataString(row.Metadata, "template")
	if templateOverride != "" && !controllerTemplateOptionContains(controllerTemplateOptions, templateOverride) {
		controllerTemplateOptions = append(controllerTemplateOptions, templatemgr.ControllerTemplateOption{
			Path:       templateOverride,
			Controller: row.Controller,
			Label:      templateOverride + " (custom)",
		})
	}
	data := formData(map[string]any{
		"ActiveNav":                   "routes",
		"Row":                         row,
		"IsNew":                       isNew,
		"BasePath":                    routesBase,
		"AllGroups":                   allGroups,
		"SelectedIDs":                 defaultGroupSelectedIDs(db, row.Groups, isNew),
		"ExtensionOptions":            extOptions,
		"ControllerOptions":           ctrlOptions,
		"PageData":                    routemeta.MetadataStringMap(row.Metadata),
		"TemplateOverride":            templateOverride,
		"ControllerTemplateOptions":   controllerTemplateOptions,
		"Categories":                  categories,
		"Tags":               tags,
		"Items":              items,
		"Users":              users,
	})
	if errMsg != "" {
		data["Error"] = errMsg
	}
	h.render(w, r, title, "admin/routes_form.html", data)
}

func extensionRouteOptions(extMgr *extensions.Manager, db *gorm.DB, selectedExt string) []extensionRouteOption {
	menuNames := extensionMenuNames(db)
	byName := make(map[string]extensionRouteOption)
	order := make([]string, 0)

	for _, rt := range extMgr.PageRuntimes() {
		opt := byName[rt.Model.Name]
		opt.Name = rt.Model.Name
		opt.Label = extensionMenuLabel(rt.Model.Name, menuNames)
		opt.Pages = extensions.RuntimePages(rt)
		byName[rt.Model.Name] = opt
		if !containsString(order, rt.Model.Name) {
			order = append(order, rt.Model.Name)
		}
	}
	for _, rt := range extMgr.EndpointRuntimes() {
		opt := byName[rt.Model.Name]
		opt.Name = rt.Model.Name
		opt.Label = extensionMenuLabel(rt.Model.Name, menuNames)
		opt.Endpoints = extensions.RuntimeEndpoints(rt)
		byName[rt.Model.Name] = opt
		if !containsString(order, rt.Model.Name) {
			order = append(order, rt.Model.Name)
		}
	}

	if selectedExt != "" {
		if _, ok := byName[selectedExt]; !ok {
			byName[selectedExt] = extensionRouteOption{
				Name:      selectedExt,
				Label:     extensionMenuLabel(selectedExt, menuNames),
				Pages:     append([]extension.PageDefinition(nil), extensions.DefaultPageDefinitions...),
				Endpoints: append([]extension.EndpointDefinition(nil), extensions.DefaultEndpointDefinitions...),
			}
			order = append(order, selectedExt)
		}
	}

	out := make([]extensionRouteOption, 0, len(byName))
	for _, name := range order {
		if opt, ok := byName[name]; ok {
			out = append(out, opt)
			delete(byName, name)
		}
	}
	for _, opt := range byName {
		out = append(out, opt)
	}
	return out
}

func containsString(list []string, value string) bool {
	for _, item := range list {
		if item == value {
			return true
		}
	}
	return false
}

func validateControllerRoute(row models.Route) error {
	action, ok := controllers.LookupAction(row.Controller, row.ControllerAction)
	if !ok || len(action.ConfigFields) == 0 {
		return nil
	}
	if strings.HasSuffix(strings.TrimSpace(row.Path), "/*") {
		return nil
	}
	meta := routemeta.MetadataStringMap(row.Metadata)
	for _, field := range action.ConfigFields {
		if !field.Required {
			continue
		}
		if strings.TrimSpace(meta[field.Name]) == "" {
			label := field.Label
			if label == "" {
				label = field.Name
			}
			return fmt.Errorf("%s is required for fixed paths without a wildcard segment", label)
		}
	}
	if row.ControllerAction == "feed" {
		kind := strings.TrimSpace(meta["feed_kind"])
		if kind == "" {
			kind = "global"
		}
		switch kind {
		case "category":
			if strings.TrimSpace(meta["category_slug"]) == "" {
				return fmt.Errorf("Category is required for category feeds on fixed paths")
			}
		case "tag":
			if strings.TrimSpace(meta["tag_slug"]) == "" {
				return fmt.Errorf("Tag is required for tag feeds on fixed paths")
			}
		case "author":
			if strings.TrimSpace(meta["author_key"]) == "" {
				return fmt.Errorf("Author is required for author feeds on fixed paths")
			}
		}
	}
	return nil
}

func controllerActionLabels(controllerID string) map[string]string {
	def, _, ok := controllers.Lookup(controllerID)
	if !ok {
		return map[string]string{}
	}
	out := make(map[string]string, len(def.Actions))
	for _, action := range def.Actions {
		out[action.ID] = action.Title
	}
	return out
}

func allControllerTemplateOptions(templateDir, frontendTheme string) ([]templatemgr.ControllerTemplateOption, error) {
	out := make([]templatemgr.ControllerTemplateOption, 0)
	for _, def := range controllers.Definitions() {
		opts, err := templatemgr.ListControllerTemplateOptions(templateDir, frontendTheme, def.ID, controllerActionLabels(def.ID))
		if err != nil {
			return nil, err
		}
		out = append(out, opts...)
	}
	return out, nil
}

func controllerTemplateOptionContains(options []templatemgr.ControllerTemplateOption, path string) bool {
	for _, opt := range options {
		if opt.Path == path {
			return true
		}
	}
	return false
}

func (h *Handler) routeDelete(w http.ResponseWriter, r *http.Request, idStr string) {
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
	var row models.Route
	if err := db.First(&row, id).Error; err != nil {
		h.notFound(w, r)
		return
	}
	if router.IsBuiltinControllerRoute(row) {
		http.Error(w, "built-in controller routes cannot be deleted", http.StatusBadRequest)
		return
	}
	wasDefault := row.IsDefault
	if err := db.Exec("DELETE FROM route_groups WHERE route_route_id = ?", id).Error; err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if err := db.Delete(&models.Route{}, id).Error; err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if wasDefault {
		if err := router.EnsureRouteDefault(db); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
	redirectList(w, r, routesBase)
}
