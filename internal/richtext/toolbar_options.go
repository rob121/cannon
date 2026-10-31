package richtext

import "strings"

// ToolbarOption describes one configurable CKEditor toolbar control.
type ToolbarOption struct {
	ID    string
	Label string
}

// AllToolbarOptions lists every control available on intro and body toolbars.
func AllToolbarOptions() []ToolbarOption {
	return []ToolbarOption{
		{ID: "heading", Label: "Heading"},
		{ID: "bold", Label: "Bold"},
		{ID: "italic", Label: "Italic"},
		{ID: "link", Label: "Link"},
		{ID: "bulletedList", Label: "Bulleted list"},
		{ID: "numberedList", Label: "Numbered list"},
		{ID: "blockQuote", Label: "Block quote"},
		{ID: "cannonMediaImage", Label: "Media library"},
		{ID: "uploadImage", Label: "Upload image (Base64)"},
		{ID: "insertImage", Label: "Insert image (Base64)"},
		{ID: "insertTable", Label: "Table"},
		{ID: "mediaEmbed", Label: "Media embed"},
		{ID: "|", Label: "Separator"},
		{ID: "sourceEditing", Label: "HTML source"},
		{ID: "undo", Label: "Undo"},
		{ID: "redo", Label: "Redo"},
	}
}

// IntroToolbarOptions is the add-button pool for intro / short fields (same as body).
func IntroToolbarOptions() []ToolbarOption {
	return AllToolbarOptions()
}

// BodyToolbarOptions is the add-button pool for body fields.
func BodyToolbarOptions() []ToolbarOption {
	return AllToolbarOptions()
}

// FormatToolbar serializes toolbar item ids for storage.
func FormatToolbar(items []string) string {
	return strings.Join(items, ", ")
}
