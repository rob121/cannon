package templatemgr

import (
	"fmt"
	"io/fs"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const (
	VersionsDir    = "versions"
	maxVersionKeep = 3
)

// File describes a template file on disk relative to the site template root.
type File struct {
	Path    string
	ModTime time.Time
	Size    int64
}

// List returns HTML template files under root, excluding versions, extension, and assets trees.
func List(root string) ([]File, error) {
	root = strings.TrimSpace(root)
	if root == "" {
		return nil, fmt.Errorf("template directory is not configured")
	}
	if err := os.MkdirAll(root, 0755); err != nil {
		return nil, err
	}

	var files []File
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			switch entry.Name() {
			case VersionsDir, "extension", "assets":
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
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		files = append(files, File{
			Path:    filepath.ToSlash(rel),
			ModTime: info.ModTime(),
			Size:    info.Size(),
		})
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Slice(files, func(i, j int) bool { return files[i].Path < files[j].Path })
	return files, nil
}

// CleanRelPath strips query fragments and decodes URL-encoded template paths.
func CleanRelPath(rel string) string {
	rel = strings.TrimSpace(rel)
	if idx := strings.IndexAny(rel, "?&"); idx >= 0 {
		rel = rel[:idx]
	}
	rel = filepath.ToSlash(strings.TrimSpace(rel))
	for {
		decoded, err := url.QueryUnescape(rel)
		if err != nil || decoded == rel {
			break
		}
		rel = decoded
	}
	return rel
}

// ValidateRelPath ensures a template path stays within the template root.
func ValidateRelPath(rel string) error {
	rel = CleanRelPath(rel)
	if rel == "" {
		return fmt.Errorf("template path is required")
	}
	if strings.HasPrefix(rel, "/") || strings.Contains(rel, "..") {
		return fmt.Errorf("invalid template path")
	}
	if !strings.HasSuffix(strings.ToLower(rel), ".html") {
		return fmt.Errorf("template path must end with .html")
	}
	return nil
}

// Save writes template content and retains up to three prior versions.
func Save(root, relPath string, content []byte) error {
	if err := ValidateRelPath(relPath); err != nil {
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
		return fmt.Errorf("write template: %w", err)
	}
	return nil
}

func writeVersion(root, relPath string, content []byte) error {
	base := filepath.Base(relPath)
	stamp := fmt.Sprintf("%d", time.Now().UnixNano())
	versionName := fmt.Sprintf("version-%s-%s", stamp, base)
	versionDir := filepath.Join(root, VersionsDir, filepath.Dir(filepath.FromSlash(relPath)))
	if err := os.MkdirAll(versionDir, 0755); err != nil {
		return err
	}
	versionPath := filepath.Join(versionDir, versionName)
	if err := os.WriteFile(versionPath, content, 0644); err != nil {
		return fmt.Errorf("write template version: %w", err)
	}
	return pruneVersions(versionDir, base)
}

func pruneVersions(versionDir, baseName string) error {
	prefix := "version-"
	suffix := "-" + baseName
	entries, err := os.ReadDir(versionDir)
	if err != nil {
		return err
	}
	var matches []string
	for _, entry := range entries {
		name := entry.Name()
		if strings.HasPrefix(name, prefix) && strings.HasSuffix(name, suffix) {
			matches = append(matches, name)
		}
	}
	if len(matches) <= maxVersionKeep {
		return nil
	}
	sort.Strings(matches)
	for _, name := range matches[:len(matches)-maxVersionKeep] {
		if err := os.Remove(filepath.Join(versionDir, name)); err != nil && !os.IsNotExist(err) {
			return err
		}
	}
	return nil
}
