package templatemgr

import (
	"fmt"
	"path"
	"sort"
	"strings"
	"time"

	"github.com/rob121/cannon/internal/templateengine"
	"github.com/rob121/cannon/internal/templatemeta"
	"github.com/rob121/cannon/internal/themes"
)

// ThemeSummary describes a theme folder for the admin browser.
type ThemeSummary = themes.Summary

// TemplateEntry is a template within a theme.
type TemplateEntry struct {
	Path       string
	Name       string
	Relative   string
	Overridden bool
	Builtin    bool
	Size       int64
	ModTime    time.Time
}

// TemplateFolder groups templates that share the same folder path within a theme.
type TemplateFolder struct {
	Key         string
	Label       string
	Entries     []TemplateEntry
	Total       int
	Overridden  int
	HasOverride bool
}

// GroupEntriesByFolder splits theme templates into root files and folder groups.
func GroupEntriesByFolder(entries []TemplateEntry, col, dir string) (root []TemplateEntry, folders []TemplateFolder) {
	grouped := map[string][]TemplateEntry{}
	for _, entry := range entries {
		key := templateEntryFolder(entry.Relative)
		if key == "" {
			root = append(root, entry)
		} else {
			grouped[key] = append(grouped[key], entry)
		}
	}
	SortEntries(root, col, dir)

	keys := make([]string, 0, len(grouped))
	for key := range grouped {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	folders = make([]TemplateFolder, 0, len(keys))
	for _, key := range keys {
		items := grouped[key]
		SortEntries(items, col, dir)
		folder := TemplateFolder{
			Key:     key,
			Label:   key,
			Entries: items,
			Total:   len(items),
		}
		for _, item := range items {
			if item.Overridden {
				folder.Overridden++
				folder.HasOverride = true
			}
		}
		folders = append(folders, folder)
	}
	return root, folders
}

func templateEntryFolder(relative string) string {
	dir := path.Dir(filepathToSlash(relative))
	if dir == "." {
		return ""
	}
	return dir
}

// ValidateTheme ensures a theme name is safe to use in URLs.
func ValidateTheme(name string) error {
	return themes.ValidateName(name)
}

// ListThemes returns installable theme folders under template_dir.
func ListThemes(root string) ([]ThemeSummary, error) {
	return themes.List(root)
}

// ThemeTemplates lists templates within a theme, merging built-in defaults for frontend/admin themes.
func ThemeTemplates(root, theme string) ([]TemplateEntry, error) {
	if err := ValidateTheme(theme); err != nil {
		return nil, err
	}
	meta, err := templatemeta.Load(themes.Root(root, theme))
	if err != nil {
		return nil, err
	}

	onDisk := map[string]File{}
	if strings.TrimSpace(root) != "" {
		if files, err := ListThemeFiles(root, theme); err == nil {
			for _, file := range files {
				onDisk[file.Path] = file
			}
		}
	}

	entries := map[string]TemplateEntry{}
	add := func(fullPath, relative string, builtin bool) {
		entry := TemplateEntry{
			Path:     fullPath,
			Name:     templateBaseName(relative),
			Relative: relative,
			Builtin:  builtin,
		}
		if file, ok := onDisk[fullPath]; ok {
			entry.Overridden = true
			entry.Size = file.Size
			entry.ModTime = file.ModTime
		}
		entries[fullPath] = entry
	}

	for _, builtinPath := range builtinPathsForTheme(theme, meta.Type) {
		relative := themes.LogicalToThemeRelative(builtinPath)
		fullPath := themes.ThemeRelativePath(theme, relative)
		add(fullPath, relative, true)
	}
	for path, file := range onDisk {
		if _, exists := entries[path]; exists {
			continue
		}
		relative := strings.TrimPrefix(path, theme+"/")
		entries[path] = TemplateEntry{
			Path:       path,
			Name:       templateBaseName(relative),
			Relative:   relative,
			Overridden: true,
			Builtin:    false,
			Size:       file.Size,
			ModTime:    file.ModTime,
		}
	}

	out := make([]TemplateEntry, 0, len(entries))
	for _, entry := range entries {
		out = append(out, entry)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Relative < out[j].Relative })
	return out, nil
}

func builtinPathsForTheme(theme, themeType string) []string {
	builtins, err := templateengine.BuiltinTemplates()
	if err != nil {
		return nil
	}
	prefix := ""
	switch {
	case theme == themes.BuiltinFrontend:
		prefix = themes.BuiltinFrontend + "/"
	case theme == themes.BuiltinAdmin:
		prefix = themes.BuiltinAdmin + "/"
	case themeAllowsFrontend(themeType):
		prefix = themes.BuiltinFrontend + "/"
	case themeAllowsAdmin(themeType):
		prefix = themes.BuiltinAdmin + "/"
	default:
		prefix = themes.BuiltinFrontend + "/"
	}
	out := make([]string, 0)
	for _, path := range builtins {
		if strings.HasPrefix(path, prefix) {
			out = append(out, path)
		}
	}
	return out
}

func themeAllowsFrontend(themeType string) bool {
	switch strings.ToLower(strings.TrimSpace(themeType)) {
	case templatemeta.TypeFrontend, templatemeta.TypeFull, "":
		return true
	default:
		return false
	}
}

func themeAllowsAdmin(themeType string) bool {
	switch strings.ToLower(strings.TrimSpace(themeType)) {
	case templatemeta.TypeBackend, templatemeta.TypeFull, "":
		return true
	default:
		return false
	}
}

// ListThemeFiles returns HTML files for one theme, excluding assets and versions.
func ListThemeFiles(root, theme string) ([]File, error) {
	root = strings.TrimSpace(root)
	if root == "" {
		return nil, fmt.Errorf("template directory is not configured")
	}
	themeRoot := themes.Root(root, theme)
	return listHTMLUnder(themeRoot, theme)
}

func listHTMLUnder(absRoot, relPrefix string) ([]File, error) {
	var files []File
	err := walkThemeHTML(absRoot, relPrefix, func(rel string, mod time.Time, size int64) error {
		files = append(files, File{Path: rel, ModTime: mod, Size: size})
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Slice(files, func(i, j int) bool { return files[i].Path < files[j].Path })
	return files, nil
}

// ThemeFromPath returns the theme name from a stored template path.
func ThemeFromPath(path string) string {
	path = filepathToSlash(path)
	parts := strings.Split(path, "/")
	if len(parts) < 2 {
		return ""
	}
	return parts[0]
}

// ValidateGroup is kept for compatibility with older callers.
func ValidateGroup(name string) error {
	return ValidateTheme(name)
}

// GroupFromPath returns the theme for a template path.
func GroupFromPath(path string) string {
	return ThemeFromPath(path)
}

// RootGroups lists themes for legacy callers.
func RootGroups(root string) ([]GroupSummary, error) {
	themesList, err := ListThemes(root)
	if err != nil {
		return nil, err
	}
	out := make([]GroupSummary, 0, len(themesList))
	for _, theme := range themesList {
		out = append(out, GroupSummary{
			Name:  theme.Name,
			Label: theme.Label,
			Type:  theme.Type,
			Status: theme.Status,
			Total: theme.Total,
		})
	}
	return out, nil
}

// GroupTemplates lists templates for a theme (legacy name).
func GroupTemplates(root, group string) ([]TemplateEntry, error) {
	return ThemeTemplates(root, group)
}

// GroupSummary describes a theme folder for admin listing.
type GroupSummary struct {
	Name       string
	Label      string
	Type       string
	Status     string
	Total      int
	Overridden int
	CustomOnly int
}

func filepathToSlash(path string) string {
	return strings.TrimPrefix(strings.ReplaceAll(path, "\\", "/"), "/")
}

func templateBaseName(path string) string {
	base := path
	if idx := strings.LastIndex(path, "/"); idx >= 0 {
		base = path[idx+1:]
	}
	return strings.TrimSuffix(base, ".html")
}

// SortGroups sorts theme summaries in place.
func SortGroups(groups []GroupSummary, col, dir string) {
	sort.Slice(groups, func(i, j int) bool {
		switch col {
		case "total":
			return sortLessInt(groups[i].Total, groups[j].Total, dir)
		case "overridden":
			return sortLessInt(groups[i].Overridden, groups[j].Overridden, dir)
		case "type":
			return sortLess(groups[i].Type, groups[j].Type, dir)
		case "status":
			return sortLess(groups[i].Status, groups[j].Status, dir)
		default:
			return sortLess(groups[i].Name, groups[j].Name, dir)
		}
	})
}

// SortEntries sorts template entries in place.
func SortEntries(entries []TemplateEntry, col, dir string) {
	sort.Slice(entries, func(i, j int) bool {
		switch col {
		case "source":
			return sortLessBool(entries[i].Builtin, entries[j].Builtin, dir)
		case "status":
			return sortLessBool(entries[i].Overridden, entries[j].Overridden, dir)
		default:
			return sortLess(entries[i].Name, entries[j].Name, dir)
		}
	})
}

func sortLess(a, b string, dir string) bool {
	if dir == "desc" {
		return a > b
	}
	return a < b
}

func sortLessInt(a, b int, dir string) bool {
	if dir == "desc" {
		return a > b
	}
	return a < b
}

func sortLessBool(a, b bool, dir string) bool {
	if dir == "desc" {
		return boolRank(a) > boolRank(b)
	}
	return boolRank(a) < boolRank(b)
}

func boolRank(v bool) int {
	if v {
		return 1
	}
	return 0
}

// ApplyMeta is a no-op kept for compatibility; metadata is loaded per theme now.
func ApplyMeta(groups []GroupSummary, _ templatemeta.PackMeta) {}
