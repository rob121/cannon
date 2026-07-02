package router

// SystemRoute describes a frontend path handled by Cannon before the site route table.
type SystemRoute struct {
	Name        string
	Path        string
	Handler     string
}

// SystemRoutes returns reserved paths that cannot be assigned in /admin/routes.
func SystemRoutes() []SystemRoute {
	return []SystemRoute{
		{
			Name:    "Admin",
			Path:    "/admin/*",
			Handler: "Cannon admin interface",
		},
		{
			Name:    "Admin Login",
			Path:    "/admin/login",
			Handler: "Admin sign-in",
		},
		{
			Name:    "Admin Assets",
			Path:    "/admin/assets/*",
			Handler: "Admin CSS, JavaScript, and icons",
		},
		{
			Name:    "Site Assets",
			Path:    "/assets/*",
			Handler: "Site asset files from the configured assets directory",
		},
		{
			Name:    "Robots",
			Path:    "/robots.txt",
			Handler: "robots.txt from SEO settings or Cannon defaults",
		},
		{
			Name:    "Theme Assets",
			Path:    "/theme/*",
			Handler: "Built-in theme CSS and icons",
		},
		{
			Name:    "Extension Data",
			Path:    "/ext/{route_hash}/*",
			Handler: "Extension /data handlers (one hash per extension route)",
		},
		{
			Name:    "Installer",
			Path:    "/install/*",
			Handler: "Installer redirect when the site is already configured",
		},
	}
}
