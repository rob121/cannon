package templatemgr

import (
	"io/fs"
	"path/filepath"
	"strings"
	"time"

	"github.com/rob121/cannon/internal/templatemeta"
)

func walkThemeHTML(absRoot, relPrefix string, fn func(rel string, mod time.Time, size int64) error) error {
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
		rel = filepath.ToSlash(rel)
		if relPrefix != "" {
			rel = filepath.ToSlash(filepath.Join(relPrefix, rel))
		}
		return fn(rel, info.ModTime(), info.Size())
	})
}
