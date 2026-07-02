package admin

import (
	"net/http"

	"github.com/rob121/cannon/extension"
	"github.com/rob121/cannon/internal/controllers"
	_ "github.com/rob121/cannon/internal/controllers/auth"
	_ "github.com/rob121/cannon/internal/controllers/content"
	"github.com/rob121/cannon/internal/extensions"
	"github.com/rob121/cannon/internal/models"
	"github.com/rob121/cannon/internal/routemeta"
	"github.com/rob121/cannon/internal/sites"
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
			http.NotFound(w, r)
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
	data := listPage(page, total, routesBase,
		"Configure URL paths and handlers.",
		"Add Route", map[string]any{"ActiveNav": "routes"})
	order := applyListSort(r, data, map[string]string{
		"name": "name", "path": "path", "type": "type", "status": "status",
	}, "name")
	db.Offset((page - 1) * pageSize).Limit(pageSize).Order(order).Find(&rows)
	data["Rows"] = rows
	h.render(w, r, "Routes", "admin/routes.html", data)
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
	if !isNew {
		if err := db.Preload("Groups").First(&row, id).Error; err != nil {
			http.NotFound(w, r)
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
				h.renderRouteForm(w, r, row, allGroups, extOptions, controllerOptions(row.Controller), isNew, "Controller and action are required.")
				return
			}
		}
		metaRaw, err := routemeta.MetadataFromForm(r.Form)
		if err != nil {
			h.renderRouteForm(w, r, row, allGroups, extOptions, controllerOptions(row.Controller), isNew, err.Error())
			return
		}
		row.Metadata = metaRaw

		if row.Type != models.RouteTypeExtension && row.Type != models.RouteTypeExtensionEndpoint {
			row.ExtensionName = ""
			row.ExtensionPageID = ""
			row.ExtensionEndpointID = ""
			row.Metadata = ""
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
		if isNew {
			saveErr = db.Create(&row).Error
		} else {
			saveErr = db.Save(&row).Error
		}
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
	data := formData(map[string]any{
		"ActiveNav":          "routes",
		"Row":                row,
		"IsNew":              isNew,
		"BasePath":           routesBase,
		"AllGroups":          allGroups,
		"SelectedIDs":        defaultGroupSelectedIDs(db, row.Groups, isNew),
		"ExtensionOptions":   extOptions,
		"ControllerOptions":  ctrlOptions,
		"PageData":           routemeta.MetadataStringMap(row.Metadata),
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

func (h *Handler) routeDelete(w http.ResponseWriter, r *http.Request, idStr string) {
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
	if err := db.Exec("DELETE FROM route_groups WHERE route_route_id = ?", id).Error; err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if err := db.Delete(&models.Route{}, id).Error; err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	redirectList(w, r, routesBase)
}
