package admin

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/rob121/cannon/internal/config"
	"github.com/rob121/cannon/internal/models"
	"gorm.io/gorm"
)

var mediaFolderSegmentPattern = regexp.MustCompile(`^[a-z0-9]+(?:[-_][a-z0-9]+)*$`)

type mediaFolderOption struct {
	Path    string
	Label   string
	Segment string
	Depth   int
}

// normalizeMediaFolder validates and normalizes a media folder path.
func normalizeMediaFolder(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	raw = strings.Trim(raw, "/")
	if raw == "" {
		return "", nil
	}
	parts := strings.Split(raw, "/")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" || part == "." || part == ".." {
			return "", fmt.Errorf("invalid folder path")
		}
		seg, err := sanitizeMediaFolderSegment(part)
		if err != nil {
			return "", err
		}
		out = append(out, seg)
	}
	return strings.Join(out, "/"), nil
}

func sanitizeMediaFolderSegment(name string) (string, error) {
	name = strings.TrimSpace(strings.ToLower(name))
	name = strings.ReplaceAll(name, " ", "-")
	if name == "" {
		return "", fmt.Errorf("folder name is required")
	}
	if !mediaFolderSegmentPattern.MatchString(name) {
		return "", fmt.Errorf("folder name may only contain letters, numbers, hyphens, and underscores")
	}
	return name, nil
}

func joinMediaFolder(parent, name string) (string, error) {
	seg, err := sanitizeMediaFolderSegment(name)
	if err != nil {
		return "", err
	}
	parent = strings.Trim(strings.TrimSpace(parent), "/")
	if parent == "" {
		return seg, nil
	}
	if _, err := normalizeMediaFolder(parent); err != nil {
		return "", err
	}
	return parent + "/" + seg, nil
}

func mediaFolderOptions(site *config.SiteConfig, db *gorm.DB) ([]mediaFolderOption, error) {
	paths, err := discoverMediaFolders(site, db)
	if err != nil {
		return nil, err
	}
	if len(paths) == 0 {
		return []mediaFolderOption{{Path: "content", Label: "content", Depth: 0}}, nil
	}
	out := make([]mediaFolderOption, 0, len(paths))
	for _, path := range paths {
		depth := strings.Count(path, "/")
		segment := path
		if idx := strings.LastIndex(path, "/"); idx >= 0 {
			segment = path[idx+1:]
		}
		out = append(out, mediaFolderOption{
			Path:    path,
			Label:   mediaFolderDisplayPath(path),
			Segment: segment,
			Depth:   depth,
		})
	}
	return out, nil
}

func discoverMediaFolders(site *config.SiteConfig, db *gorm.DB) ([]string, error) {
	seen := map[string]struct{}{"content": {}}
	if site != nil {
		root := filepath.Join(site.AssetsDir, "media")
		_ = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
			if err != nil || !d.IsDir() {
				return nil
			}
			if path == root {
				return nil
			}
			rel, relErr := filepath.Rel(root, path)
			if relErr != nil {
				return nil
			}
			seen[filepath.ToSlash(rel)] = struct{}{}
			return nil
		})
	}
	if db != nil {
		var dbFolders []string
		db.Model(&models.MediaAsset{}).Distinct("folder").Pluck("folder", &dbFolders)
		for _, folder := range dbFolders {
			folder = strings.TrimSpace(folder)
			if folder != "" {
				seen[folder] = struct{}{}
			}
		}
	}
	paths := make([]string, 0, len(seen))
	for folder := range seen {
		paths = append(paths, folder)
	}
	sort.Strings(paths)
	return paths, nil
}

func mediaChildFolderNav(all []string, parent string, db *gorm.DB) []mediaFolderNav {
	parent = strings.Trim(strings.TrimSpace(parent), "/")
	seen := map[string]struct{}{}
	out := make([]mediaFolderNav, 0)
	for _, path := range all {
		childPath, childName, ok := directMediaChildFolder(path, parent)
		if !ok {
			continue
		}
		if _, exists := seen[childPath]; exists {
			continue
		}
		seen[childPath] = struct{}{}
		var count int64
		if db != nil {
			db.Model(&models.MediaAsset{}).Where("folder = ?", childPath).Count(&count)
		}
		out = append(out, mediaFolderNav{
			Name:  childPath,
			Label: childName,
			Count: count,
			URL:   mediaFolderURL(childPath),
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

func directMediaChildFolder(path, parent string) (fullPath, segment string, ok bool) {
	path = strings.Trim(strings.TrimSpace(path), "/")
	if path == "" {
		return "", "", false
	}
	if parent == "" {
		if idx := strings.Index(path, "/"); idx >= 0 {
			return path[:idx], path[:idx], true
		}
		return path, path, true
	}
	prefix := parent + "/"
	if !strings.HasPrefix(path, prefix) {
		return "", "", false
	}
	rest := strings.TrimPrefix(path, prefix)
	if rest == "" {
		return "", "", false
	}
	if idx := strings.Index(rest, "/"); idx >= 0 {
		segment = rest[:idx]
	} else {
		segment = rest
	}
	return parent + "/" + segment, segment, true
}

func mediaFolderBreadcrumbs(folder string) []map[string]string {
	folder = strings.Trim(strings.TrimSpace(folder), "/")
	crumbs := []map[string]string{{"Label": "All Files", "URL": mediaBase}}
	if folder == "" {
		return crumbs
	}
	parts := strings.Split(folder, "/")
	path := ""
	for i, part := range parts {
		if i > 0 {
			path += "/"
		}
		path += part
		crumbs = append(crumbs, map[string]string{
			"Label": part,
			"URL":   mediaFolderURL(path),
		})
	}
	return crumbs
}

func mediaFolderDisplayPath(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return "Root"
	}
	return strings.ReplaceAll(path, "/", " / ")
}

func ensureMediaFolder(site *config.SiteConfig, folder string) error {
	if site == nil {
		return fmt.Errorf("site not found")
	}
	folder, err := normalizeMediaFolder(folder)
	if err != nil {
		return err
	}
	if folder == "" {
		return fmt.Errorf("folder name is required")
	}
	dest := filepath.Join(site.AssetsDir, "media", filepath.FromSlash(folder))
	return os.MkdirAll(dest, 0o755)
}
