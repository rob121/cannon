package server

import (
	"context"
	"html/template"
	"net/http"
	"strings"
	"time"

	"github.com/rob121/cannon/internal/config"
	cms "github.com/rob121/cannon/internal/content"
	"github.com/rob121/cannon/internal/groups"
	"github.com/rob121/cannon/internal/models"
	"github.com/rob121/cannon/internal/routepath"
	"github.com/rob121/cannon/internal/router"
	"github.com/rob121/cannon/internal/settings"
	"github.com/rob121/cannon/internal/sites"
	"github.com/rob121/cannon/internal/templateengine"
)

type frontendFuncOpts struct {
	Request   *http.Request
	Site      *config.SiteConfig
	RenderCtx func() context.Context
}

func frontendFuncMap(opts frontendFuncOpts) template.FuncMap {
	r := opts.Request
	site := opts.Site
	renderCtx := opts.RenderCtx
	if renderCtx == nil {
		renderCtx = func() context.Context {
			if r == nil {
				return context.Background()
			}
			return r.Context()
		}
	}
	return template.FuncMap{
		"menu": func(name string) ([]map[string]any, error) {
			return router.MenuData(renderCtx(), name)
		},
		"siteName": func() string {
			if site != nil {
				return site.Name
			}
			return ""
		},
		"isOffline": func() bool {
			if r == nil {
				return false
			}
			offline, err := settings.SiteOffline(r.Context())
			return err == nil && offline
		},
		"homeURL": func() string {
			return sites.DefaultRoutePath(renderCtx())
		},
		"controllerURL": func(controller, action string) string {
			if r == nil {
				return ""
			}
			return routepath.Controller(r.Context(), controller, action)
		},
		"controllerURLFor": func(controller, action, suffix string) string {
			if r == nil {
				return ""
			}
			return routepath.ControllerWithSuffix(r.Context(), controller, action, suffix)
		},
		"isDefaultRoute": func() bool {
			return router.IsDefaultRouteContext(renderCtx())
		},
		"isRoute": func(controller, action string) bool {
			return router.IsRouteContext(renderCtx(), controller, action)
		},
		"isCurrentPath": func(path string) bool {
			return templateengine.IsCurrentPath(r, path)
		},
		"routeName": func() string {
			if route, ok := router.RouteFromContext(renderCtx()); ok {
				return route.Name
			}
			return ""
		},
		"routePath": func() string {
			if route, ok := router.RouteFromContext(renderCtx()); ok {
				return route.Path
			}
			return ""
		},
		"routeController": func() string {
			if route, ok := router.RouteFromContext(renderCtx()); ok {
				return route.Controller
			}
			return ""
		},
		"routeAction": func() string {
			if route, ok := router.RouteFromContext(renderCtx()); ok {
				return route.Action
			}
			return ""
		},
		"showRouteTitle": func() bool {
			if route, ok := router.MatchedRoute(renderCtx()); ok {
				return router.ShowTitleForRoute(route)
			}
			return true
		},
		"siteMetaDescription": func() string {
			if r == nil {
				return ""
			}
			desc, _ := settings.SiteMetaDescription(r.Context())
			return desc
		},
		"siteMetaKeywords": func() string {
			if r == nil {
				return ""
			}
			keywords, _ := settings.SiteMetaKeywords(r.Context())
			return keywords
		},
		"siteOGTitle": func() string {
			if r == nil {
				return ""
			}
			title, _ := settings.SiteOGTitle(r.Context())
			return title
		},
		"siteOGImage": func() string {
			if r == nil {
				return ""
			}
			image, _ := settings.SiteOGImage(r.Context())
			return image
		},
		"siteTwitterCard": func() string {
			if r == nil {
				return "summary_large_image"
			}
			card, _ := settings.SiteTwitterCard(r.Context())
			return card
		},
		"siteTwitterSite": func() string {
			if r == nil {
				return ""
			}
			handle, _ := settings.SiteTwitterSite(r.Context())
			return handle
		},
		"siteTwitterCreator": func() string {
			if r == nil {
				return ""
			}
			creator, _ := settings.SiteTwitterCreator(r.Context())
			return creator
		},
		"siteHeadExtra": func() template.HTML {
			if r == nil {
				return ""
			}
			raw, _ := settings.SiteHeadExtra(r.Context())
			return template.HTML(raw)
		},
		"year": func() int {
			return time.Now().Year()
		},
		"userName": templateengine.UserName,
		"formatDate": func(t *time.Time, layout string) string {
			return templateengine.FormatDate(t, layout)
		},
		"itemDate": func(item models.Item, layout string) string {
			return templateengine.ItemDate(item, layout)
		},
		"itemDateISO": templateengine.ItemDateISO,
		"itemHasDate": templateengine.ItemHasDate,
		"excerpt":     templateengine.Excerpt,
		"themeAsset":  templateengine.ThemeAssetURL,
		"absoluteURL": func(path string) string {
			return templateengine.AbsoluteURL(r, path)
		},
		"setting": func(section, key string) string {
			if r == nil {
				return ""
			}
			return templateengine.PublicSetting(r.Context(), section, key)
		},
		"settingBool": func(section, key string) bool {
			if r == nil {
				return false
			}
			return templateengine.PublicSettingBool(r.Context(), section, key)
		},
		"richText": func(src string) template.HTML {
			html, err := cms.RichTextToHTML(src)
			if err != nil {
				return template.HTML("")
			}
			return template.HTML(html)
		},
		"items": func(mode string, limit int) ([]models.Item, error) {
			if r == nil {
				return nil, nil
			}
			viewerGroups, err := groups.ViewerGroupIDs(r.Context())
			if err != nil {
				return nil, err
			}
			opts := cms.ListOptions{Page: 1, Limit: limit}
			switch strings.TrimSpace(strings.ToLower(mode)) {
			case "featured":
				opts.Featured = true
			case "popular":
				return cms.PopularItems(r.Context(), viewerGroups, limit)
			}
			items, _, err := cms.ListItems(r.Context(), viewerGroups, opts)
			return items, err
		},
		"itemURL": func(slug string) string {
			if r == nil {
				return ""
			}
			return cms.ItemURLForContext(r.Context(), slug)
		},
		"categoryURL": func(slug string) string {
			if r == nil {
				return ""
			}
			return cms.CategoryURLForContext(r.Context(), slug)
		},
		"tagURL": func(slug string) string {
			if r == nil {
				return ""
			}
			return cms.TagURLForContext(r.Context(), slug)
		},
		"authorURL": func(key string) string {
			if r == nil {
				return ""
			}
			return cms.AuthorURLForContext(r.Context(), key)
		},
		"searchURL": func(query string) string {
			if r == nil {
				return ""
			}
			return cms.SearchURLForContext(r.Context(), query)
		},
		"featuredURL": func() string {
			if r == nil {
				return ""
			}
			return cms.FeaturedURLForContext(r.Context())
		},
		"categories": func() ([]models.Category, error) {
			if r == nil {
				return nil, nil
			}
			return cms.CategoryTree(r.Context())
		},
		"tags": func() ([]models.Tag, error) {
			if r == nil {
				return nil, nil
			}
			return cms.ListTags(r.Context())
		},
		"tagCloud": func(limit int) ([]cms.TagCount, error) {
			if r == nil {
				return nil, nil
			}
			viewerGroups, err := groups.ViewerGroupIDs(r.Context())
			if err != nil {
				return nil, err
			}
			return cms.TagCloud(r.Context(), viewerGroups, limit)
		},
		"commentCount": func(itemID uint) (int64, error) {
			if r == nil {
				return 0, nil
			}
			return cms.CommentCount(r.Context(), itemID)
		},
		"fieldOptions": func(raw string) []cms.FieldOption {
			return cms.ParseFieldConfig(raw).Options
		},
		"fieldValueContains": cms.FieldValueContains,
		"safeHTML": func(s string) template.HTML {
			return template.HTML(s)
		},
		"formatDateTimeLocal": func(t *time.Time) string {
			return templateengine.FormatDate(t, "2006-01-02T15:04")
		},
	}
}
