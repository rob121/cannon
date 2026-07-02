package extension

// BlockDefinition describes one renderable block exposed by an extension.
type BlockDefinition struct {
	ID     string       `json:"id"`
	Title  string       `json:"title,omitempty"`
	Spaces []string     `json:"spaces,omitempty"`
	Fields []BlockField `json:"fields,omitempty"`
}

// BlockListResponse is returned from GET /block.
type BlockListResponse struct {
	Blocks []BlockDefinition `json:"blocks"`
}

// BlockField describes one admin-editable metadata value for a block placement.
type BlockField struct {
	Name     string             `json:"name"`
	Label    string             `json:"label,omitempty"`
	Type     string             `json:"type,omitempty"`
	Help     string             `json:"help,omitempty"`
	Required bool               `json:"required,omitempty"`
	Options  []BlockFieldOption `json:"options,omitempty"`
}

// BlockFieldOption describes one selectable block field value.
type BlockFieldOption struct {
	Value string `json:"value"`
	Label string `json:"label,omitempty"`
}

// BlockHandler renders one registered block item.
type BlockHandler func(item string, req WireRequest) WireResponse

// BlockListProvider returns block definitions for GET /block.
type BlockListProvider func() ([]BlockDefinition, error)
