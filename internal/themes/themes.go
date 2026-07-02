package themes

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/rob121/cannon/internal/templatemeta"
)

const (
	BuiltinFrontend = "default"
	BuiltinAdmin    = "admin"
)

var reservedNames = map[string]bool{
	"versions":  true,
	"extension": true,
}

// Summary describes an installable theme folder under template_dir.
type Summary struct {
	Name        string
	Label       string
	Type        string
	Status      string
	Description string
	Version     string
	Author      string
	Total       int
}

// Selection holds the active frontend and admin theme names.
type Selection struct {
	Frontend string
	Admin    string
}

// Normalize returns built-in theme ids when empty.
func (s Selection) Normalize() Selection {
	out := s
	if strings.TrimSpace(out.Frontend) == "" {
		out.Frontend = BuiltinFrontend
	}
	if strings.TrimSpace(out.Admin) == "" {
		out.Admin = BuiltinAdmin
	}
	return out
}

// IsBuiltinFrontend reports whether the frontend theme is the embedded default pack.
func IsBuiltinFrontend(name string) bool {
	name = strings.TrimSpace(name)
	return name == "" || name == BuiltinFrontend
}

// IsBuiltinAdmin reports whether the admin theme is the embedded admin pack.
func IsBuiltinAdmin(name string) bool {
	name = strings.TrimSpace(name)
	return name == "" || name == BuiltinAdmin
}

// ValidateName ensures a theme folder name is safe.
func ValidateName(name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return fmt.Errorf("theme name is required")
	}
	if strings.Contains(name, "/") || strings.Contains(name, "..") {
		return fmt.Errorf("invalid theme name")
	}
	if reservedNames[name] {
		return fmt.Errorf("reserved theme name %q", name)
	}
	return nil
}

// Root returns the absolute theme directory path.
func Root(templateDir, name string) string {
	return themesRoot(templateDir, name)
}

func themesRoot(templateDir, name string) string {
	templateDir = strings.TrimSpace(templateDir)
	name = strings.TrimSpace(name)
	if templateDir == "" || name == "" {
		return ""
	}
	return filepath.Join(templateDir, name)
}

// AssetsDir returns the theme assets directory path.
func AssetsDir(templateDir, name string) string {
	root := Root(templateDir, name)
	if root == "" {
		return ""
	}
	return filepath.Join(root, "assets")
}

// List discovers theme folders under template_dir.
func List(templateDir string) ([]Summary, error) {
	templateDir = strings.TrimSpace(templateDir)
	if templateDir == "" {
		return nil, nil
	}
	entries, err := os.ReadDir(templateDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	out := make([]Summary, 0)
	for _, entry := range entries {
		if !entry.IsDir() || reservedNames[entry.Name()] {
			continue
		}
		if err := ValidateName(entry.Name()); err != nil {
			continue
		}
		summary, err := summarize(templateDir, entry.Name())
		if err != nil {
			return nil, err
		}
		if summary.Total == 0 {
			if _, err := os.Stat(filepath.Join(themesRoot(templateDir, entry.Name()), templatemeta.FileName)); err != nil {
				continue
			}
		}
		out = append(out, summary)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}

func summarize(templateDir, name string) (Summary, error) {
	meta, err := templatemeta.Load(Root(templateDir, name))
	if err != nil {
		return Summary{}, err
	}
	total, err := countHTML(Root(templateDir, name))
	if err != nil {
		return Summary{}, err
	}
	label := strings.TrimSpace(meta.Name)
	if label == "" {
		label = name
	}
	return Summary{
		Name:        name,
		Label:       label,
		Type:        meta.Type,
		Status:      meta.Status,
		Description: meta.Description,
		Version:     meta.Version,
		Author:      meta.Author,
		Total:       total,
	}, nil
}

func countHTML(root string) (int, error) {
	total := 0
	err := walkHTML(root, func(string, time.Time, int64) error {
		total++
		return nil
	})
	return total, err
}

func walkHTML(absRoot string, fn func(rel string, mod time.Time, size int64) error) error {
	absRoot = strings.TrimSpace(absRoot)
	if absRoot == "" {
		return nil
	}
	return filepath.WalkDir(absRoot, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			switch entry.Name() {
			case "assets", templatemeta.VersionsDirName:
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(strings.ToLower(entry.Name()), ".html") {
			return nil
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(absRoot, path)
		if err != nil {
			return err
		}
		return fn(filepath.ToSlash(rel), info.ModTime(), info.Size())
	})
}

// FrontendOptions returns dropdown values for frontend theme selection.
func FrontendOptions(templateDir string) ([]string, error) {
	opts := []string{BuiltinFrontend}
	themes, err := List(templateDir)
	if err != nil {
		return nil, err
	}
	for _, theme := range themes {
		if themeAllowsFrontend(theme.Type) && theme.Status == templatemeta.StatusActive {
			opts = append(opts, theme.Name)
		}
	}
	return opts, nil
}

// AdminOptions returns dropdown values for admin theme selection.
func AdminOptions(templateDir string) ([]string, error) {
	opts := []string{BuiltinAdmin}
	themes, err := List(templateDir)
	if err != nil {
		return nil, err
	}
	for _, theme := range themes {
		if themeAllowsAdmin(theme.Type) && theme.Status == templatemeta.StatusActive {
			opts = append(opts, theme.Name)
		}
	}
	return opts, nil
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

// LogicalToBuiltin converts an engine logical path to the embedded builtin path.
func LogicalToBuiltin(logical string) string {
	logical = filepath.ToSlash(strings.TrimPrefix(logical, "/"))
	if strings.HasPrefix(logical, "admin/") || strings.HasPrefix(logical, "default/") {
		return logical
	}
	return ""
}

// LogicalToThemeRelative strips the default/ or admin/ prefix for theme disk layout.
func LogicalToThemeRelative(logical string) string {
	logical = filepath.ToSlash(strings.TrimPrefix(logical, "/"))
	switch {
	case strings.HasPrefix(logical, "default/"):
		return strings.TrimPrefix(logical, "default/")
	case strings.HasPrefix(logical, "admin/"):
		return strings.TrimPrefix(logical, "admin/")
	default:
		return logical
	}
}

// ThemeRelativePath joins a theme name and relative template path for storage under template_dir.
func ThemeRelativePath(theme, relative string) string {
	relative = filepath.ToSlash(strings.TrimPrefix(relative, "/"))
	if relative == "" {
		return theme
	}
	return filepath.ToSlash(filepath.Join(theme, relative))
}

// DiskPathForLogical maps a logical engine path to an on-disk path under template_dir for the active theme.
func DiskPathForLogical(templateDir, theme, logical string, admin bool) string {
	if admin {
		if IsBuiltinAdmin(theme) {
			return ""
		}
		rel := LogicalToThemeRelative(logical)
		return filepath.Join(templateDir, theme, filepath.FromSlash(rel))
	}
	if IsBuiltinFrontend(theme) {
		return ""
	}
	rel := LogicalToThemeRelative(logical)
	return filepath.Join(templateDir, theme, filepath.FromSlash(rel))
}

// Active reports whether a theme pack is active according to its template.json.
func Active(templateDir, name string) bool {
	if IsBuiltinFrontend(name) || IsBuiltinAdmin(name) {
		return true
	}
	meta, err := templatemeta.Load(Root(templateDir, name))
	if err != nil {
		return false
	}
	return meta.Active()
}

// OptionLabel returns a human-friendly label for configuration dropdowns.
func OptionLabel(name string) string {
	switch strings.TrimSpace(name) {
	case "", BuiltinFrontend:
		return "Built-in (default)"
	case BuiltinAdmin:
		return "Built-in (admin)"
	default:
		return name
	}
}
