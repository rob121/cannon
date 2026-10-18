package templatemgr

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/rob121/cannon/internal/templatemeta"
	"github.com/rob121/cannon/internal/themes"
)

// LegacyReport describes a detected pre-theme template_dir layout.
type LegacyReport struct {
	HasRootMeta   bool
	HasDefaultDir bool
	HasAdminDir   bool
	DefaultFiles  int
	AdminFiles    int
	Notes         []string
}

var themeNameSlugRE = regexp.MustCompile(`[^a-z0-9]+`)

// DetectLegacy reports whether template_dir still uses the old pack layout
// (root template.json and/or default|/admin trees without per-theme metadata).
func DetectLegacy(root string) (LegacyReport, bool) {
	var report LegacyReport
	root = strings.TrimSpace(root)
	if root == "" {
		return report, false
	}
	if _, err := os.Stat(filepath.Join(root, templatemeta.FileName)); err == nil {
		report.HasRootMeta = true
	}
	if isLegacyOverrideDir(filepath.Join(root, themes.BuiltinFrontend)) {
		report.HasDefaultDir = true
		report.DefaultFiles, _ = countHTMLFiles(filepath.Join(root, themes.BuiltinFrontend))
	}
	if isLegacyOverrideDir(filepath.Join(root, themes.BuiltinAdmin)) {
		report.HasAdminDir = true
		report.AdminFiles, _ = countHTMLFiles(filepath.Join(root, themes.BuiltinAdmin))
	}
	if report.HasRootMeta {
		report.Notes = append(report.Notes, "Root template.json will be archived into the new theme metadata.")
	}
	if report.HasDefaultDir {
		report.Notes = append(report.Notes, fmt.Sprintf("Legacy default/ tree (%d HTML files) will become a frontend theme.", report.DefaultFiles))
	}
	if report.HasAdminDir {
		report.Notes = append(report.Notes, fmt.Sprintf("Legacy admin/ tree (%d HTML files) will become a backend theme.", report.AdminFiles))
	}
	return report, report.HasRootMeta || report.HasDefaultDir || report.HasAdminDir
}

// MigrateLegacy moves legacy default/admin trees into named theme folders.
// frontendName is required when a default/ tree exists; adminName when admin/ exists.
// If only one tree exists, a single theme of the appropriate type is created.
func MigrateLegacy(root, frontendName, adminName string) (frontendTheme, adminTheme string, err error) {
	report, ok := DetectLegacy(root)
	if !ok {
		return "", "", fmt.Errorf("no legacy template layout detected")
	}
	root = strings.TrimSpace(root)
	frontendName = sanitizeThemeName(frontendName)
	adminName = sanitizeThemeName(adminName)

	rootMeta, _ := templatemeta.Load(root)

	needFrontend := report.HasDefaultDir || (report.HasRootMeta && !report.HasAdminDir)
	needAdmin := report.HasAdminDir

	if needFrontend {
		if frontendName == "" {
			frontendName = "site"
		}
		if err := themes.ValidateName(frontendName); err != nil {
			return "", "", err
		}
		if dirExists(filepath.Join(root, frontendName)) {
			return "", "", fmt.Errorf("theme folder %q already exists", frontendName)
		}
	}
	if needAdmin {
		if adminName == "" {
			if frontendName != "" {
				adminName = frontendName + "-admin"
			} else {
				adminName = "admin-custom"
			}
		}
		if err := themes.ValidateName(adminName); err != nil {
			return "", "", err
		}
		if needFrontend && adminName == frontendName {
			return "", "", fmt.Errorf("frontend and admin theme names must differ")
		}
		if dirExists(filepath.Join(root, adminName)) {
			return "", "", fmt.Errorf("theme folder %q already exists", adminName)
		}
	}

	if needFrontend {
		dest := filepath.Join(root, frontendName)
		if err := os.MkdirAll(dest, 0o755); err != nil {
			return "", "", err
		}
		if report.HasDefaultDir {
			if err := copyTree(filepath.Join(root, themes.BuiltinFrontend), dest); err != nil {
				return "", "", err
			}
		}
		_ = os.MkdirAll(filepath.Join(dest, "assets"), 0o755)
		meta := rootMeta
		if meta.Name == "" {
			meta.Name = frontendName
		}
		if needAdmin {
			meta.Type = templatemeta.TypeFrontend
		} else {
			meta.Type = templatemeta.TypeFull
		}
		meta.Groups = nil
		meta.Status = templatemeta.StatusActive
		if err := templatemeta.Save(dest, meta); err != nil {
			return "", "", err
		}
		frontendTheme = frontendName
	}

	if needAdmin {
		dest := filepath.Join(root, adminName)
		if err := os.MkdirAll(dest, 0o755); err != nil {
			return "", "", err
		}
		if err := copyTree(filepath.Join(root, themes.BuiltinAdmin), dest); err != nil {
			return "", "", err
		}
		_ = os.MkdirAll(filepath.Join(dest, "assets"), 0o755)
		meta := templatemeta.DefaultPackMeta()
		meta.Name = adminName
		meta.Type = templatemeta.TypeBackend
		meta.Status = templatemeta.StatusActive
		if rootMeta.Author != "" {
			meta.Author = rootMeta.Author
		}
		if err := templatemeta.Save(dest, meta); err != nil {
			return "", "", err
		}
		adminTheme = adminName
	}

	if report.HasDefaultDir {
		if err := os.RemoveAll(filepath.Join(root, themes.BuiltinFrontend)); err != nil {
			return frontendTheme, adminTheme, fmt.Errorf("remove legacy default/: %w", err)
		}
	}
	if report.HasAdminDir {
		if err := os.RemoveAll(filepath.Join(root, themes.BuiltinAdmin)); err != nil {
			return frontendTheme, adminTheme, fmt.Errorf("remove legacy admin/: %w", err)
		}
	}
	if report.HasRootMeta {
		_ = os.Remove(filepath.Join(root, templatemeta.FileName))
	}

	if frontendTheme == "" && adminTheme == "" {
		return "", "", fmt.Errorf("migration produced no themes")
	}
	return frontendTheme, adminTheme, nil
}

func isLegacyOverrideDir(dir string) bool {
	info, err := os.Stat(dir)
	if err != nil || !info.IsDir() {
		return false
	}
	// Modern theme folder already has its own template.json.
	if _, err := os.Stat(filepath.Join(dir, templatemeta.FileName)); err == nil {
		return false
	}
	n, _ := countHTMLFiles(dir)
	return n > 0
}

func countHTMLFiles(root string) (int, error) {
	total := 0
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			switch d.Name() {
			case VersionsDir, "extension", "assets":
				if path != root {
					return filepath.SkipDir
				}
			}
			return nil
		}
		if strings.HasSuffix(strings.ToLower(d.Name()), ".html") {
			total++
		}
		return nil
	})
	return total, err
}

func sanitizeThemeName(name string) string {
	name = strings.ToLower(strings.TrimSpace(name))
	name = themeNameSlugRE.ReplaceAllString(name, "-")
	name = strings.Trim(name, "-")
	return name
}

func dirExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

func copyTree(src, dst string) error {
	src = strings.TrimSpace(src)
	dst = strings.TrimSpace(dst)
	return filepath.WalkDir(src, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		if rel == "." {
			return nil
		}
		target := filepath.Join(dst, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		return copyFile(path, target, info.Mode())
	})
}
