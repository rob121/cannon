package blocks

import (
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/rob121/cannon/internal/templateengine"
	"github.com/rob121/cannon/internal/templatemgr"
	"github.com/rob121/cannon/internal/themes"
)

var (
	spaceDoubleQuoted = regexp.MustCompile(`\{\{\s*space\s+"([^"]+)"\s*\}\}`)
	spaceBackticked   = regexp.MustCompile("\\{\\{\\s*space\\s+`([^`]+)`\\s*\\}\\}")
)

var defaultKnownSpaces = []string{
	"header", "sidebar", "footer",
	"admin-dashboard-top", "admin-dashboard-widgets", "admin-dashboard-main",
}

// ExtractSpacesFromSource returns literal {{space "name"}} / {{space `name`}} names from template source.
func ExtractSpacesFromSource(src string) []string {
	seen := map[string]struct{}{}
	var out []string
	add := func(name string) {
		name = strings.TrimSpace(name)
		if name == "" {
			return
		}
		if _, ok := seen[name]; ok {
			return
		}
		seen[name] = struct{}{}
		out = append(out, name)
	}
	for _, m := range spaceDoubleQuoted.FindAllStringSubmatch(src, -1) {
		if len(m) > 1 {
			add(m[1])
		}
	}
	for _, m := range spaceBackticked.FindAllStringSubmatch(src, -1) {
		if len(m) > 1 {
			add(m[1])
		}
	}
	return out
}

// ListKnownSpaces discovers space names from embedded + active theme templates
// (frontend and admin), merges default layout spaces and any extra names
// (e.g. DistinctSpaces from DB).
func ListKnownSpaces(templateDir, frontendTheme, adminTheme string, extras ...[]string) ([]string, error) {
	seen := map[string]struct{}{}
	out := make([]string, 0)

	add := func(name string) {
		name = strings.TrimSpace(name)
		if name == "" {
			return
		}
		if _, ok := seen[name]; ok {
			return
		}
		seen[name] = struct{}{}
		out = append(out, name)
	}
	for _, name := range defaultKnownSpaces {
		add(name)
	}

	builtins, err := templateengine.BuiltinTemplates()
	if err != nil {
		return nil, err
	}
	for _, path := range builtins {
		if !strings.HasPrefix(path, themes.BuiltinFrontend+"/") && !strings.HasPrefix(path, themes.BuiltinAdmin+"/") {
			continue
		}
		raw, err := templateengine.ReadBuiltin(path)
		if err != nil {
			continue
		}
		for _, name := range ExtractSpacesFromSource(string(raw)) {
			add(name)
		}
	}

	frontendTheme = strings.TrimSpace(frontendTheme)
	adminTheme = strings.TrimSpace(adminTheme)
	themesToScan := []string{themes.BuiltinFrontend, themes.BuiltinAdmin}
	addTheme := func(theme string) {
		theme = strings.TrimSpace(theme)
		if theme == "" {
			return
		}
		for _, existing := range themesToScan {
			if existing == theme {
				return
			}
		}
		themesToScan = append(themesToScan, theme)
	}
	addTheme(frontendTheme)
	addTheme(adminTheme)
	for _, theme := range themesToScan {
		files, err := templatemgr.ListThemeFiles(templateDir, theme)
		if err != nil {
			continue
		}
		themeRoot := themes.Root(templateDir, theme)
		for _, file := range files {
			rel := strings.TrimPrefix(file.Path, theme+"/")
			diskPath := filepath.Join(themeRoot, filepath.FromSlash(rel))
			raw, err := os.ReadFile(diskPath)
			if err != nil {
				continue
			}
			for _, name := range ExtractSpacesFromSource(string(raw)) {
				add(name)
			}
		}
	}

	for _, group := range extras {
		for _, name := range group {
			add(name)
		}
	}

	sort.Strings(out)
	return out, nil
}
