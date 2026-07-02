package templatemgr

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/rob121/cannon/internal/templateengine"
)

// Read loads template content from a theme folder or built-in defaults.
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

	theme := ThemeFromPath(relPath)
	relative := strings.TrimPrefix(relPath, theme+"/")
	builtinPath := logicalBuiltinPath(root, theme, relative)
	builtin, err := templateengine.ReadBuiltin(builtinPath)
	if err != nil {
		return "", "", false, fmt.Errorf("template not found")
	}
	return builtin, "builtin:" + builtinPath, true, nil
}

// AvailableBuiltins returns built-in templates not yet copied into a theme.
func AvailableBuiltins(root, theme string) ([]string, error) {
	entries, err := ThemeTemplates(root, theme)
	if err != nil {
		return nil, err
	}
	out := make([]string, 0)
	for _, entry := range entries {
		if entry.Builtin && !entry.Overridden {
			out = append(out, entry.Path)
		}
	}
	return out, nil
}
