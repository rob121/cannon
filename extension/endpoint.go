package extension

// EndpointDefinition describes one data/action handler exposed by an extension.
type EndpointDefinition struct {
	ID     string          `json:"id"`
	Title  string          `json:"title,omitempty"`
	Fields []EndpointField `json:"fields,omitempty"`
}

// EndpointListResponse is returned from GET /endpoint.
type EndpointListResponse struct {
	Endpoints []EndpointDefinition `json:"endpoints"`
}

// EndpointField describes one admin-editable metadata value for a route placement.
type EndpointField = BlockField

// EndpointFieldOption describes one selectable endpoint field value.
type EndpointFieldOption = BlockFieldOption

// EndpointHandler handles one registered endpoint item.
type EndpointHandler func(item string, req WireRequest) WireResponse

// EndpointListProvider returns endpoint definitions for GET /endpoint.
type EndpointListProvider func() ([]EndpointDefinition, error)
