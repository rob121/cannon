package templatemgr

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/rob121/cannon/internal/templateengine"
	"github.com/rob121/cannon/internal/templatemeta"
	"github.com/rob121/cannon/internal/themes"
)

// Override copies a built-in template into a theme folder.
func Override(root, relPath string) error {
	if err := ValidateRelPath(relPath); err != nil {
		return err
	}
	root = strings.TrimSpace(root)
	if root == "" {
		return fmt.Errorf("template directory is not configured")
	}

	theme := ThemeFromPath(relPath)
	if theme == "" {
		return fmt.Errorf("template path must include a theme folder")
	}
	if err := ValidateTheme(theme); err != nil {
		return err
	}

	target := filepath.Join(root, filepath.FromSlash(relPath))
	if _, err := os.Stat(target); err == nil {
		return fmt.Errorf("template already overridden")
	}

	relative := strings.TrimPrefix(relPath, theme+"/")
	builtinPath := logicalBuiltinPath(root, theme, relative)
	content, err := templateengine.ReadBuiltin(builtinPath)
	if err != nil {
		return fmt.Errorf("template not found: %s", relPath)
	}
	return Save(root, relPath, []byte(content))
}

// CanRevertOverride reports whether an on-disk override can be removed to restore the built-in template.
func CanRevertOverride(root, relPath string) bool {
	if err := ValidateRelPath(relPath); err != nil {
		return false
	}
	root = strings.TrimSpace(root)
	if root == "" {
		return false
	}
	theme := ThemeFromPath(relPath)
	if theme == "" {
		return false
	}
	relative := strings.TrimPrefix(relPath, theme+"/")
	builtinPath := logicalBuiltinPath(root, theme, relative)
	if _, err := templateengine.ReadBuiltin(builtinPath); err != nil {
		return false
	}
	target := filepath.Join(root, filepath.FromSlash(relPath))
	if _, err := os.Stat(target); err != nil {
		return false
	}
	return true
}

// RevertOverride removes an on-disk template override so the built-in copy is used again.
func RevertOverride(root, relPath string) error {
	if err := ValidateRelPath(relPath); err != nil {
		return err
	}
	if !CanRevertOverride(root, relPath) {
		return fmt.Errorf("template override not found")
	}
	target := filepath.Join(strings.TrimSpace(root), filepath.FromSlash(relPath))
	if err := os.Remove(target); err != nil {
		return fmt.Errorf("remove template override: %w", err)
	}
	return nil
}

func logicalBuiltinPath(root, theme, relative string) string {
	switch theme {
	case themes.BuiltinAdmin:
		return themes.BuiltinAdmin + "/" + relative
	case themes.BuiltinFrontend:
		return themes.BuiltinFrontend + "/" + relative
	}
	meta, _ := templatemeta.Load(themes.Root(root, theme))
	if themeAllowsAdmin(meta.Type) && !themeAllowsFrontend(meta.Type) {
		return themes.BuiltinAdmin + "/" + relative
	}
	return themes.BuiltinFrontend + "/" + relative
}
