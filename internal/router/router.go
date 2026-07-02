package router

import (
	"context"
	"errors"
	"fmt"
	"html/template"
	"net/http"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/rob121/cannon/internal/controllers"
	"github.com/rob121/cannon/internal/extensions"
	"github.com/rob121/cannon/internal/groups"
	"github.com/rob121/cannon/internal/hooks"
	"github.com/rob121/cannon/internal/models"
	"github.com/rob121/cannon/internal/routemeta"
	"github.com/rob121/cannon/internal/settings"
	"github.com/rob121/cannon/internal/sites"
	"github.com/rob121/cannon/internal/templateengine"
	"gorm.io/gorm"
)

type matchKind int

const (
	kindExact matchKind = iota
	kindWildcard
	kindRegex
)

type matchedRoute struct {
	Route models.Route
	Kind  matchKind
	Score int
}

// Dispatcher resolves dynamic routes for a site.
type Dispatcher struct {
	ext *extensions.Manager
	tpl *templateengine.Engine
}

func NewDispatcher(ext *extensions.Manager, tpl *templateengine.Engine) *Dispatcher {
	return &Dispatcher{ext: ext, tpl: tpl}
}

func (d *Dispatcher) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	userCtx, _ := userContext(ctx)
	if handled, err := d.ext.ServeExtensionData(ctx, w, r, userCtx); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	} else if handled {
		return
	}

	if offline, err := settings.SiteOffline(r.Context()); err == nil && offline {
		d.renderMaintenance(w, r)
		return
	}

	beforeArgs := hooks.RequestArgs(r)
	if _, err := hooks.Fire(ctx, hooks.OnBeforeRoute, beforeArgs); err != nil {
		http.Error(w, err.Error(), http.StatusForbidden)
		return
	}

	db, err := sites.DB(ctx)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	route, ok, err := d.match(ctx, db, r.URL.Path)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if !ok {
		if d.noViewableRoutes(ctx, db) {
			d.renderNoRoutes(w, r, http.StatusNotFound)
		} else {
			http.NotFound(w, r)
		}
		return
	}

	afterArgs := hooks.RequestArgs(r)
	for k, v := range routeHookArgs(route) {
		afterArgs[k] = v
	}
	if _, err := hooks.Fire(ctx, hooks.OnAfterRoute, afterArgs); err != nil {
		http.Error(w, err.Error(), http.StatusForbidden)
		return
	}

	switch route.Type {
	case models.RouteTypeURL:
		target := route.Target
		if target == "" {
			http.NotFound(w, r)
			return
		}
		http.Redirect(w, r, target, http.StatusFound)
	case models.RouteTypeLocalFile:
		site, _ := sites.FromContext(ctx)
		path := filepath.Join(site.AssetsDir, route.Target)
		http.ServeFile(w, r, path)
	case models.RouteTypeExtension:
		userCtx, _ := userContext(ctx)
		pageData, err := routemeta.MetadataMap(route.Metadata)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		prepareArgs := map[string]any{
			"content_type": "extension_page",
			"extension":    route.ExtensionName,
			"page_item":    route.ExtensionPageID,
			"page_data":    pageData,
		}
		for k, v := range routeHookArgs(route) {
			prepareArgs[k] = v
		}
		if _, err := hooks.Fire(ctx, hooks.OnContentPrepare, prepareArgs); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		html, err := d.ext.RenderPage(ctx, route.ExtensionName, route.ExtensionPageID, r, userCtx, pageData)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		displayArgs := map[string]any{
			"title":   route.Name,
			"content": html,
			"layout":  "default/layout.html",
			"page":    "default/page.html",
		}
		for k, v := range routeHookArgs(route) {
			displayArgs[k] = v
		}
		displayOut, err := hooks.Fire(ctx, hooks.OnContentBeforeDisplay, displayArgs)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		title := route.Name
		if v, ok := displayOut["title"].(string); ok && v != "" {
			title = v
		}
		if v, ok := displayOut["content"].(string); ok {
			html = v
		}
		data := map[string]any{"Title": title, "Content": template.HTML(html)}
		d.tpl.SetHookContext(ctx)
		if err := d.tpl.Render(w, "default/layout.html", "default/page.html", data); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		d.tpl.SetHookContext(nil)
	case models.RouteTypeExtensionEndpoint:
		userCtx, _ := userContext(ctx)
		endpointData, err := routemeta.MetadataMap(route.Metadata)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		out, err := d.ext.InvokeEndpoint(ctx, route.ExtensionName, route.ExtensionEndpointID, r, userCtx, endpointData)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		extensions.WriteHTTPResponse(w, out)
	case models.RouteTypeController:
		userCtx, _ := userContext(ctx)
		_ = userCtx
		viewerGroups, err := groups.ViewerGroupIDs(ctx)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		d.tpl.SetHookContext(ctx)
		if err := controllers.Dispatch(w, r, route, d.tpl, viewerGroups); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		d.tpl.SetHookContext(nil)
	default:
		http.NotFound(w, r)
	}
}

func (d *Dispatcher) match(ctx context.Context, db *gorm.DB, path string) (models.Route, bool, error) {
	viewerGroups, err := groups.ViewerGroupIDs(ctx)
	if err != nil {
		return models.Route{}, false, err
	}

	var routes []models.Route
	if err := db.Preload("Groups").Where("status = ?", models.StatusActive).Find(&routes).Error; err != nil {
		return models.Route{}, false, err
	}

	var best *matchedRoute
	for _, route := range routes {
		if !groups.CanView(viewerGroups, route.Groups) {
			continue
		}
		kind, score, ok := matchPath(route.Path, path)
		if !ok {
			continue
		}
		candidate := &matchedRoute{Route: route, Kind: kind, Score: score}
		if best == nil || candidate.Score > best.Score || (candidate.Score == best.Score && candidate.Kind < best.Kind) {
			best = candidate
		}
	}
	if best == nil {
		return models.Route{}, false, nil
	}
	return best.Route, true, nil
}

func (d *Dispatcher) noViewableRoutes(ctx context.Context, db *gorm.DB) bool {
	viewerGroups, err := groups.ViewerGroupIDs(ctx)
	if err != nil {
		return false
	}
	var routes []models.Route
	if err := db.Preload("Groups").Where("status = ?", models.StatusActive).Find(&routes).Error; err != nil {
		return false
	}
	for _, route := range routes {
		if groups.CanView(viewerGroups, route.Groups) {
			return false
		}
	}
	return true
}

func (d *Dispatcher) renderMaintenance(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusServiceUnavailable)
	data := map[string]any{"Title": "Site Offline"}
	if err := d.tpl.Render(w, "default/layout.html", "default/maintenance.html", data); err != nil {
		http.Error(w, "Site is temporarily offline.", http.StatusServiceUnavailable)
	}
}

func (d *Dispatcher) renderNoRoutes(w http.ResponseWriter, r *http.Request, status int) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	data := map[string]any{"Title": "Configure your site"}
	if err := d.tpl.RenderFragment(w, "default/noroutes.html", data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func matchPath(pattern, path string) (matchKind, int, bool) {
	if pattern == path {
		return kindExact, 1000 + len(pattern), true
	}
	if strings.HasSuffix(pattern, "/*") {
		prefix := strings.TrimSuffix(pattern, "/*")
		if path == prefix || strings.HasPrefix(path, prefix+"/") {
			return kindWildcard, 500 + len(prefix), true
		}
	}
	if strings.HasPrefix(pattern, "/") && strings.ContainsAny(pattern, "[]()^$.*+?\\|") {
		re, err := regexp.Compile("^" + pattern + "$")
		if err == nil && re.MatchString(path) {
			return kindRegex, 100 + len(pattern), true
		}
	}
	return kindExact, 0, false
}

type userContextKey struct{}

func WithUserContext(ctx context.Context, data map[string]any) context.Context {
	return context.WithValue(ctx, userContextKey{}, data)
}

func UserContext(ctx context.Context) (map[string]any, bool) {
	v, ok := ctx.Value(userContextKey{}).(map[string]any)
	return v, ok
}

func userContext(ctx context.Context) (map[string]any, bool) {
	return UserContext(ctx)
}

// MenuData loads menu items for template rendering.
func MenuData(ctx context.Context, menuName string) ([]map[string]string, error) {
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
			return []map[string]string{}, nil
		}
		return nil, err
	}
	out := make([]map[string]string, 0, len(menu.Items))
	for _, item := range menu.Items {
		if !groups.CanView(viewerGroups, item.Groups) {
			continue
		}
		href := "#"
		if item.Route != nil && item.Route.Status == models.StatusActive {
			href = item.Route.Path
		}
		out = append(out, map[string]string{
			"Name":   item.Name,
			"Href":   href,
			"Class":  item.Class,
			"Target": item.Target,
		})
	}
	return out, nil
}

// EnsureDefaultRoute seeds built-in controller routes when missing.
func EnsureDefaultRoute(db *gorm.DB) error {
	publicID, err := groups.PublicGroupID(db)
	if err != nil {
		return err
	}
	var publicGroup models.Group
	if err := db.First(&publicGroup, publicID).Error; err != nil {
		return err
	}

	seeds := []models.Route{
		{Name: "Home", Path: "/", Type: models.RouteTypeController, Status: models.StatusActive, Controller: "content", ControllerAction: "index"},
		{Name: "Content Category", Path: "/content/category/*", Type: models.RouteTypeController, Status: models.StatusActive, Controller: "content", ControllerAction: "category"},
		{Name: "Content Item", Path: "/content/item/*", Type: models.RouteTypeController, Status: models.StatusActive, Controller: "content", ControllerAction: "item"},
		{Name: "Content Tag", Path: "/content/tag/*", Type: models.RouteTypeController, Status: models.StatusActive, Controller: "content", ControllerAction: "tag"},
		{Name: "Content Author", Path: "/content/author/*", Type: models.RouteTypeController, Status: models.StatusActive, Controller: "content", ControllerAction: "author"},
		{Name: "Content Search", Path: "/content/search", Type: models.RouteTypeController, Status: models.StatusActive, Controller: "content", ControllerAction: "search"},
		{Name: "Content Feed", Path: "/content/feed/*", Type: models.RouteTypeController, Status: models.StatusActive, Controller: "content", ControllerAction: "feed"},
		{Name: "Content Create", Path: "/content/edit/new", Type: models.RouteTypeController, Status: models.StatusActive, Controller: "content", ControllerAction: "edit-new"},
		{Name: "Content Edit", Path: "/content/edit/*", Type: models.RouteTypeController, Status: models.StatusActive, Controller: "content", ControllerAction: "edit"},
		{Name: "Login", Path: "/login", Type: models.RouteTypeController, Status: models.StatusActive, Controller: "auth", ControllerAction: "login"},
		{Name: "Logout", Path: "/logout", Type: models.RouteTypeController, Status: models.StatusActive, Controller: "auth", ControllerAction: "logout"},
		{Name: "Verify Account", Path: "/account/verify/*", Type: models.RouteTypeController, Status: models.StatusActive, Controller: "auth", ControllerAction: "verify"},
		{Name: "Verification Pending", Path: "/account/verify/resend", Type: models.RouteTypeController, Status: models.StatusActive, Controller: "auth", ControllerAction: "verify-resend"},
		{Name: "Reset Password", Path: "/account/reset-password", Type: models.RouteTypeController, Status: models.StatusActive, Controller: "auth", ControllerAction: "reset-request"},
		{Name: "Reset Password Submit", Path: "/account/reset-password/*", Type: models.RouteTypeController, Status: models.StatusActive, Controller: "auth", ControllerAction: "reset-submit"},
	}

	for _, seed := range seeds {
		var existing models.Route
		err := db.Where("path = ?", seed.Path).First(&existing).Error
		if err == nil {
			if existing.Type == models.RouteTypeController && (existing.Controller == "" || existing.ControllerAction == "") {
				existing.Controller = seed.Controller
				existing.ControllerAction = seed.ControllerAction
				existing.Type = models.RouteTypeController
				if err := db.Save(&existing).Error; err != nil {
					return err
				}
			}
			continue
		}
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		row := seed
		if err := db.Create(&row).Error; err != nil {
			return err
		}
		if err := db.Model(&row).Association("Groups").Replace([]models.Group{publicGroup}); err != nil {
			return err
		}
	}
	return nil
}

func NotFoundHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, fmt.Sprintf("not found: %s", r.URL.Path), http.StatusNotFound)
	})
}
