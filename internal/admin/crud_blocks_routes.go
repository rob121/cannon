package admin

import (
	"sort"
	"strings"

	"github.com/rob121/cannon/internal/models"
	"github.com/rob121/cannon/internal/router"
)

type blockRouteSection struct {
	Key    string
	Label  string
	Desc   string
	Routes []models.Route
}

func buildBlockRouteSections(routes []models.Route) []blockRouteSection {
	var (
		defaultRoutes []models.Route
		sitePages     []models.Route
		content       []models.Route
		authAccount   []models.Route
		extensions    []models.Route
		urlsFiles     []models.Route
		other         []models.Route
	)
	for _, route := range routes {
		switch {
		case route.IsDefault:
			defaultRoutes = append(defaultRoutes, route)
		case route.Type == models.RouteTypeExtension || route.Type == models.RouteTypeExtensionEndpoint:
			extensions = append(extensions, route)
		case route.Type == models.RouteTypeURL || route.Type == models.RouteTypeLocalFile:
			urlsFiles = append(urlsFiles, route)
		case route.Type == models.RouteTypeController:
			if router.IsBuiltinControllerRoute(route) {
				if isAuthAccountRoute(route) {
					authAccount = append(authAccount, route)
				} else {
					content = append(content, route)
				}
			} else {
				sitePages = append(sitePages, route)
			}
		default:
			other = append(other, route)
		}
	}

	sortBlockRoutes(defaultRoutes)
	sortBlockRoutes(sitePages)
	sortBlockRoutes(content)
	sortBlockRoutes(authAccount)
	sortBlockRoutes(extensions)
	sortBlockRoutes(urlsFiles)
	sortBlockRoutes(other)

	sections := make([]blockRouteSection, 0, 7)
	if len(defaultRoutes) > 0 {
		sections = append(sections, blockRouteSection{
			Key:    "default",
			Label:  "Default Site Route",
			Desc:   "The page visitors see at the site root and when no other route matches.",
			Routes: defaultRoutes,
		})
	}
	if len(sitePages) > 0 {
		sections = append(sections, blockRouteSection{
			Key:    "site",
			Label:  "Site Pages",
			Desc:   "Custom controller routes you added under Routes, such as a home page or landing page.",
			Routes: sitePages,
		})
	}
	if len(content) > 0 {
		sections = append(sections, blockRouteSection{
			Key:    "content",
			Label:  "Content Routes",
			Desc:   "Built-in content controller paths for categories, items, search, and feeds.",
			Routes: content,
		})
	}
	if len(authAccount) > 0 {
		sections = append(sections, blockRouteSection{
			Key:    "auth",
			Label:  "Authentication & Account",
			Desc:   "Built-in sign-in, sign-out, verification, and password reset routes.",
			Routes: authAccount,
		})
	}
	if len(extensions) > 0 {
		sections = append(sections, blockRouteSection{
			Key:    "extensions",
			Label:  "Extensions",
			Desc:   "Routes served by installed extensions and extension endpoints.",
			Routes: extensions,
		})
	}
	if len(urlsFiles) > 0 {
		sections = append(sections, blockRouteSection{
			Key:    "urls",
			Label:  "URLs & Files",
			Desc:   "Redirects, external URLs, and local file routes.",
			Routes: urlsFiles,
		})
	}
	if len(other) > 0 {
		sections = append(sections, blockRouteSection{
			Key:    "other",
			Label:  "Other Routes",
			Routes: other,
		})
	}
	return sections
}

func isAuthAccountRoute(route models.Route) bool {
	path := strings.TrimSpace(route.Path)
	return strings.HasPrefix(path, "/auth/") || strings.HasPrefix(path, "/account/")
}

func sortBlockRoutes(routes []models.Route) {
	sort.Slice(routes, func(i, j int) bool {
		if routes[i].IsDefault != routes[j].IsDefault {
			return routes[i].IsDefault
		}
		if routes[i].Name != routes[j].Name {
			return routes[i].Name < routes[j].Name
		}
		return routes[i].Path < routes[j].Path
	})
}
