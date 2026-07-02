package extension

// PageDefinition describes one renderable page exposed by an extension.
type PageDefinition struct {
	ID     string      `json:"id"`
	Title  string      `json:"title,omitempty"`
	Fields []PageField `json:"fields,omitempty"`
}

// PageListResponse is returned from GET /page.
type PageListResponse struct {
	Pages []PageDefinition `json:"pages"`
}

// PageField describes one admin-editable metadata value for a route placement.
type PageField = BlockField

// PageFieldOption describes one selectable page field value.
type PageFieldOption = BlockFieldOption

// PageHandler renders one registered page item.
type PageHandler func(item string, req WireRequest) WireResponse

// PageListProvider returns page definitions for GET /page.
type PageListProvider func() ([]PageDefinition, error)
