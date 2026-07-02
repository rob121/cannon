package templatemgr

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/rob121/cannon/internal/templateengine"
)

// Read loads template content from disk or built-in defaults.
func Read(root, relPath string) (content string, source string, fromBuiltin bool, err error) {
	if err := ValidateRelPath(relPath); err != nil {
		return "", "", false, err
	}
	root = strings.TrimSpace(root)
	if root == "" {
		return "", "", false, fmt.Errorf("template directory is not configured")
	}

	target := filepath.Join(root, filepath.FromSlash(relPath))
	if raw, err := os.ReadFile(target); err == nil {
		return string(raw), relPath, false, nil
	}

	builtin, err := templateengine.ReadBuiltin(relPath)
	if err != nil {
		return "", "", false, fmt.Errorf("template not found")
	}
	return builtin, "builtin:" + relPath, true, nil
}

// AvailableBuiltins returns built-in templates not yet overridden on disk.
func AvailableBuiltins(root string) ([]string, error) {
	builtins, err := templateengine.BuiltinTemplates()
	if err != nil {
		return nil, err
	}
	onDisk := map[string]bool{}
	if root != "" {
		files, err := List(root)
		if err != nil {
			return nil, err
		}
		for _, file := range files {
			onDisk[file.Path] = true
		}
	}
	out := make([]string, 0)
	for _, name := range builtins {
		if !onDisk[name] {
			out = append(out, name)
		}
	}
	return out, nil
}
