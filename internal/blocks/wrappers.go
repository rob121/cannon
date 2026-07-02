package blocks

import (
	"sort"
	"strings"

	"github.com/rob121/cannon/internal/templateengine"
	"github.com/rob121/cannon/internal/templatemgr"
	"github.com/rob121/cannon/internal/themes"
)

const blockWrappersDir = "partials/blocks/"

// WrapperOption is a selectable block wrapper template for admin forms.
type WrapperOption struct {
	Path  string
	Label string
}

var blockBodyTemplates = map[string]struct{}{
	LoginBlockTemplate:          {},
	MenuVerticalBlockTemplate:   {},
	MenuHorizontalBlockTemplate: {},
}

// ListWrapperTemplates returns block wrapper templates from embedded default
// and the configured frontend theme on disk.
func ListWrapperTemplates(templateDir, frontendTheme string) ([]WrapperOption, error) {
	seen := map[string]struct{}{}
	out := make([]WrapperOption, 0)

	add := func(fullPath string) {
		fullPath = strings.TrimSpace(fullPath)
		if fullPath == "" || !isBlockWrapperPath(fullPath) {
			return
		}
		if _, skip := blockBodyTemplates[fullPath]; skip {
			return
		}
		if _, ok := seen[fullPath]; ok {
			return
		}
		seen[fullPath] = struct{}{}
		out = append(out, WrapperOption{
			Path:  fullPath,
			Label: wrapperLabel(fullPath),
		})
	}

	builtins, err := templateengine.BuiltinTemplates()
	if err != nil {
		return nil, err
	}
	for _, p := range builtins {
		add(p)
	}

	themesToScan := []string{themes.BuiltinFrontend}
	frontendTheme = strings.TrimSpace(frontendTheme)
	if frontendTheme != "" && frontendTheme != themes.BuiltinFrontend {
		themesToScan = append(themesToScan, frontendTheme)
	}
	for _, theme := range themesToScan {
		files, err := templatemgr.ListThemeFiles(templateDir, theme)
		if err != nil {
			continue
		}
		for _, file := range files {
			add(file.Path)
		}
	}

	sort.Slice(out, func(i, j int) bool { return out[i].Path < out[j].Path })
	return out, nil
}

func isBlockWrapperPath(fullPath string) bool {
	fullPath = strings.TrimSpace(fullPath)
	if !strings.HasSuffix(strings.ToLower(fullPath), ".html") {
		return false
	}
	parts := strings.Split(fullPath, "/")
	if len(parts) < 4 {
		return false
	}
	return parts[1] == "partials" && parts[2] == "blocks"
}

func wrapperLabel(fullPath string) string {
	parts := strings.Split(fullPath, "/")
	if len(parts) < 2 {
		return fullPath
	}
	theme := parts[0]
	name := strings.TrimSuffix(parts[len(parts)-1], ".html")
	return theme + " / " + name
}
