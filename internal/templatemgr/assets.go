package templatemgr

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/rob121/cannon/internal/themes"
)

// AssetEntry describes one file under a theme assets directory.
type AssetEntry struct {
	Path     string
	Relative string
	Name     string
	Ext      string
	Editable bool
	Size     int64
	ModTime  time.Time
}

// AssetFolder groups asset files by subdirectory.
type AssetFolder struct {
	Key    string
	Label  string
	Entries []AssetEntry
	Total  int
}

var editableAssetExtensions = map[string]string{
	"css":  "css",
	"js":   "javascript",
	"json": "javascript",
	"html": "htmlmixed",
	"htm":  "htmlmixed",
	"svg":  "xml",
	"xml":  "xml",
	"md":   "markdown",
	"txt":  "text",
	"map":  "javascript",
}

// EditorModeForAsset returns the CodeMirror mode for an asset extension.
func EditorModeForAsset(ext string) string {
	ext = strings.ToLower(strings.TrimPrefix(strings.TrimSpace(ext), "."))
	if mode, ok := editableAssetExtensions[ext]; ok {
		return mode
	}
	return "text"
}

// IsEditableAsset reports whether an asset can be edited in the text editor.
func IsEditableAsset(ext string) bool {
	ext = strings.ToLower(strings.TrimPrefix(strings.TrimSpace(ext), "."))
	_, ok := editableAssetExtensions[ext]
	return ok
}

// AssetPath builds the stored path for a theme asset relative to template_dir.
func AssetPath(theme, relative string) string {
	relative = strings.TrimPrefix(filepathToSlash(relative), "/")
	return theme + "/assets/" + relative
}

// ValidateAssetRelPath ensures a path points at a theme asset file.
func ValidateAssetRelPath(rel string) error {
	rel = CleanRelPath(rel)
	if rel == "" {
		return fmt.Errorf("asset path is required")
	}
	if strings.HasPrefix(rel, "/") || strings.Contains(rel, "..") {
		return fmt.Errorf("invalid asset path")
	}
	parts := strings.Split(rel, "/")
	if len(parts) < 3 || parts[1] != "assets" {
		return fmt.Errorf("asset path must be theme/assets/…")
	}
	if err := ValidateTheme(parts[0]); err != nil {
		return err
	}
	base := parts[len(parts)-1]
	if base == "" || strings.HasSuffix(base, "/") {
		return fmt.Errorf("asset filename is required")
	}
	if !IsEditableAsset(filepath.Ext(base)) {
		return fmt.Errorf("asset type %q is not editable in the admin editor", filepath.Ext(base))
	}
	return nil
}

// ValidateAssetStoragePath ensures a path points at any file under theme/assets/.
func ValidateAssetStoragePath(rel string) error {
	rel = CleanRelPath(rel)
	if rel == "" {
		return fmt.Errorf("asset path is required")
	}
	if strings.HasPrefix(rel, "/") || strings.Contains(rel, "..") {
		return fmt.Errorf("invalid asset path")
	}
	parts := strings.Split(rel, "/")
	if len(parts) < 3 || parts[1] != "assets" {
		return fmt.Errorf("asset path must be theme/assets/…")
	}
	if err := ValidateTheme(parts[0]); err != nil {
		return err
	}
	base := parts[len(parts)-1]
	if base == "" || strings.HasSuffix(base, "/") {
		return fmt.Errorf("asset filename is required")
	}
	return nil
}

// ThemeFromAssetPath returns the theme name from a stored asset path.
func ThemeFromAssetPath(path string) string {
	return ThemeFromPath(path)
}

// AssetRelative returns the path under assets/ from a stored asset path.
func AssetRelative(path string) string {
	path = filepathToSlash(path)
	const marker = "/assets/"
	if idx := strings.Index(path, marker); idx >= 0 {
		return strings.TrimPrefix(path[idx+len(marker):], "/")
	}
	return ""
}

// ListThemeAssets returns files under template_dir/{theme}/assets.
func ListThemeAssets(root, theme string) ([]AssetEntry, error) {
	if err := ValidateTheme(theme); err != nil {
		return nil, err
	}
	root = strings.TrimSpace(root)
	if root == "" {
		return nil, fmt.Errorf("template directory is not configured")
	}
	assetsRoot := filepath.Join(themes.Root(root, theme), "assets")
	if err := os.MkdirAll(assetsRoot, 0755); err != nil {
		return nil, err
	}

	var entries []AssetEntry
	err := filepath.WalkDir(assetsRoot, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(assetsRoot, path)
		if err != nil {
			return err
		}
		rel = filepathToSlash(rel)
		ext := strings.ToLower(strings.TrimPrefix(filepath.Ext(entry.Name()), "."))
		entries = append(entries, AssetEntry{
			Path:     AssetPath(theme, rel),
			Relative: rel,
			Name:     entry.Name(),
			Ext:      ext,
			Editable: IsEditableAsset(ext),
			Size:     info.Size(),
			ModTime:  info.ModTime(),
		})
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Relative < entries[j].Relative })
	return entries, nil
}

// GroupAssetsByFolder splits assets into root files and folder groups.
func GroupAssetsByFolder(entries []AssetEntry, col, dir string) (root []AssetEntry, folders []AssetFolder) {
	grouped := map[string][]AssetEntry{}
	for _, entry := range entries {
		key := assetEntryFolder(entry.Relative)
		if key == "" {
			root = append(root, entry)
		} else {
			grouped[key] = append(grouped[key], entry)
		}
	}
	sortAssetEntries(root, col, dir)

	keys := make([]string, 0, len(grouped))
	for key := range grouped {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	folders = make([]AssetFolder, 0, len(keys))
	for _, key := range keys {
		items := grouped[key]
		sortAssetEntries(items, col, dir)
		folders = append(folders, AssetFolder{
			Key:     key,
			Label:   key,
			Entries: items,
			Total:   len(items),
		})
	}
	return root, folders
}

func assetEntryFolder(relative string) string {
	dir := filepath.Dir(filepathToSlash(relative))
	if dir == "." {
		return ""
	}
	return dir
}

func sortAssetEntries(entries []AssetEntry, col, dir string) {
	sort.Slice(entries, func(i, j int) bool {
		switch col {
		case "type":
			return sortLess(entries[i].Ext, entries[j].Ext, dir)
		case "editable":
			return sortLessBool(entries[i].Editable, entries[j].Editable, dir)
		default:
			return sortLess(entries[i].Name, entries[j].Name, dir)
		}
	})
}

// ReadAsset loads editable asset content from disk.
func ReadAsset(root, relPath string) (string, error) {
	if err := ValidateAssetRelPath(relPath); err != nil {
		return "", err
	}
	root = strings.TrimSpace(root)
	if root == "" {
		return "", fmt.Errorf("template directory is not configured")
	}
	target := filepath.Join(root, filepath.FromSlash(relPath))
	raw, err := os.ReadFile(target)
	if err != nil {
		return "", fmt.Errorf("asset not found")
	}
	return string(raw), nil
}

// SaveAsset writes asset content and retains prior versions.
func SaveAsset(root, relPath string, content []byte) error {
	if err := ValidateAssetRelPath(relPath); err != nil {
		return err
	}
	root = strings.TrimSpace(root)
	if root == "" {
		return fmt.Errorf("template directory is not configured")
	}

	target := filepath.Join(root, filepath.FromSlash(relPath))
	if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
		return err
	}

	if existing, err := os.ReadFile(target); err == nil {
		if err := writeVersion(root, relPath, existing); err != nil {
			return err
		}
	}

	if err := os.WriteFile(target, content, 0644); err != nil {
		return fmt.Errorf("write asset: %w", err)
	}
	return nil
}

// DeleteAsset removes a theme asset file from disk.
func DeleteAsset(root, relPath string) error {
	if err := ValidateAssetStoragePath(relPath); err != nil {
		return err
	}
	root = strings.TrimSpace(root)
	if root == "" {
		return fmt.Errorf("template directory is not configured")
	}
	target := filepath.Join(root, filepath.FromSlash(relPath))
	if err := os.Remove(target); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("asset not found")
		}
		return fmt.Errorf("delete asset: %w", err)
	}
	return nil
}
