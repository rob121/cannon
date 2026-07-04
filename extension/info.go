package extension

// Info describes extension identity shown in /meta and admin defaults.
type Info struct {
	Name          string
	Title         string
	Description   string
	Version       string
	UpdateURLBase string
	AdminMenuName string
}

type metaResponse struct {
	Name          string `json:"name"`
	Title         string `json:"title,omitempty"`
	Description   string `json:"description,omitempty"`
	Version       string `json:"version"`
	UpdateURLBase string `json:"update_url_base"`
	RouteHash     string `json:"route_hash,omitempty"`
}

type capabilitiesResponse struct {
	Capabilities map[string]string  `json:"capabilities"`
	Permissions  []PermissionDef    `json:"permissions,omitempty"`
	Defaults     capabilityDefaults `json:"defaults"`
}

// TemplateDefinition describes one embedded HTML template an extension ships.
type TemplateDefinition struct {
	Path         string `json:"path"`
	OverridePath string `json:"override_path"`
	Size         int64  `json:"size,omitempty"`
}

// TemplateListResponse is returned by GET /templates.
type TemplateListResponse struct {
	Templates []TemplateDefinition `json:"templates"`
}

// TemplateSourceResponse is returned by GET /templates/{path}.
type TemplateSourceResponse struct {
	Path         string `json:"path"`
	OverridePath string `json:"override_path"`
	Content      string `json:"content"`
}

type capabilityDefaults struct {
	Admin adminDefaults `json:"admin"`
}

type adminDefaults struct {
	MenuName string `json:"menu_name"`
}
