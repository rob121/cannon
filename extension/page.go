package extension

// PageDefinition describes one renderable page exposed by an extension.
type PageDefinition struct {
	ID     string      `json:"id"`
	Title  string      `json:"title,omitempty"`
	Fields []PageField `json:"fields,omitempty"`
}

// MountSpec describes a recommended extension mount route (resolver-driven subpages).
type MountSpec struct {
	ID              string            `json:"id"`
	Title           string            `json:"title,omitempty"`
	Path            string            `json:"path"`
	DefaultPageData map[string]string `json:"default_page_data,omitempty"`
	Resolver        bool              `json:"resolver,omitempty"`
}

// PageRouteSpec describes a fixed extension page placement (one page id per path).
type PageRouteSpec struct {
	PageID string `json:"page_id"`
	Path   string `json:"path"`
	Name   string `json:"name,omitempty"`
}

// PageListResponse is returned from GET /page.
type PageListResponse struct {
	Pages  []PageDefinition `json:"pages"`
	Mounts []MountSpec      `json:"mounts,omitempty"`
	Routes []PageRouteSpec  `json:"routes,omitempty"`
}

// PageResolveResponse is returned from POST /page/resolve.
type PageResolveResponse struct {
	PageItem      string `json:"page_item"`
	StatusCode    int    `json:"status_code,omitempty"`
	RedirectURL   string `json:"redirect_url,omitempty"`
	TitleOverride string `json:"title_override,omitempty"`
}

// PageResolveHandler maps a mount request to a page definition id.
type PageResolveHandler func(req WireRequest) PageResolveResponse

// PageField describes one admin-editable metadata value for a route placement.
type PageField = BlockField

// PageFieldOption describes one selectable page field value.
type PageFieldOption = BlockFieldOption

// PageHandler renders one registered page item.
type PageHandler func(item string, req WireRequest) WireResponse

// PageListProvider returns page definitions for GET /page.
type PageListProvider func() ([]PageDefinition, error)
