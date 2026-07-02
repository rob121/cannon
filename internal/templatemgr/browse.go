package templatemgr

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/rob121/cannon/internal/templateengine"
)

// GroupSummary describes a top-level template folder such as admin or default.
type GroupSummary struct {
	Name        string
	Total       int
	Overridden  int
	CustomOnly  int
}

// TemplateEntry is a template within a group.
type TemplateEntry struct {
	Path       string
	Name       string
	Overridden bool
	Builtin    bool
	Size       int64
	ModTime    time.Time
}

// ValidateGroup ensures a template group name is safe to use in URLs.
func ValidateGroup(name string) error {
	name = strings.TrimSpace(name)
	if name == "" || strings.Contains(name, "/") || strings.Contains(name, "..") {
		return fmt.Errorf("invalid template group")
	}
	return nil
}

// RootGroups returns top-level template folders from built-ins and on-disk files.
func RootGroups(root string) ([]GroupSummary, error) {
	builtins, err := templateengine.BuiltinTemplates()
	if err != nil {
		return nil, err
	}

	onDisk := map[string]File{}
	if strings.TrimSpace(root) != "" {
		if files, err := List(root); err == nil {
			for _, file := range files {
				onDisk[file.Path] = file
			}
		}
	}

	groups := map[string]*GroupSummary{}
	ensure := func(name string) *GroupSummary {
		if g, ok := groups[name]; ok {
			return g
		}
		g := &GroupSummary{Name: name}
		groups[name] = g
		return g
	}

	builtinSet := map[string]bool{}
	for _, path := range builtins {
		builtinSet[path] = true
		group, ok := groupName(path)
		if !ok {
			continue
		}
		g := ensure(group)
		g.Total++
		if _, ok := onDisk[path]; ok {
			g.Overridden++
		}
	}
	for path := range onDisk {
		group, ok := groupName(path)
		if !ok {
			continue
		}
		g := ensure(group)
		if builtinSet[path] {
			continue
		}
		g.Total++
		g.Overridden++
		g.CustomOnly++
	}

	out := make([]GroupSummary, 0, len(groups))
	for _, g := range groups {
		out = append(out, *g)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}

// GroupTemplates lists templates within a top-level folder.
func GroupTemplates(root, group string) ([]TemplateEntry, error) {
	if err := ValidateGroup(group); err != nil {
		return nil, err
	}

	builtins, err := templateengine.BuiltinTemplates()
	if err != nil {
		return nil, err
	}

	onDisk := map[string]File{}
	if strings.TrimSpace(root) != "" {
		if files, err := List(root); err == nil {
			for _, file := range files {
				onDisk[file.Path] = file
			}
		}
	}

	entries := map[string]TemplateEntry{}
	add := func(path string, builtin bool) {
		g, ok := groupName(path)
		if !ok || g != group {
			return
		}
		entry := TemplateEntry{
			Path:    path,
			Name:    templateBaseName(path),
			Builtin: builtin,
		}
		if file, ok := onDisk[path]; ok {
			entry.Overridden = true
			entry.Size = file.Size
			entry.ModTime = file.ModTime
		}
		entries[path] = entry
	}

	for _, path := range builtins {
		add(path, true)
	}
	for path, file := range onDisk {
		g, ok := groupName(path)
		if !ok || g != group {
			continue
		}
		if _, exists := entries[path]; exists {
			continue
		}
		entries[path] = TemplateEntry{
			Path:       path,
			Name:       templateBaseName(path),
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
	sort.Slice(out, func(i, j int) bool { return out[i].Path < out[j].Path })
	return out, nil
}

// GroupFromPath returns the top-level group for a template path.
func GroupFromPath(path string) string {
	group, ok := groupName(path)
	if !ok {
		return ""
	}
	return group
}

func groupName(path string) (string, bool) {
	path = filepathToSlash(path)
	parts := strings.Split(path, "/")
	if len(parts) < 2 {
		return "", false
	}
	return parts[0], true
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

// SortGroups sorts template group summaries in place.
func SortGroups(groups []GroupSummary, col, dir string) {
	sort.Slice(groups, func(i, j int) bool {
		switch col {
		case "total":
			return sortLessInt(groups[i].Total, groups[j].Total, dir)
		case "overridden":
			return sortLessInt(groups[i].Overridden, groups[j].Overridden, dir)
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
