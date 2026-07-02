package templatemgr

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/rob121/cannon/internal/templateengine"
)

// Override copies a built-in template into the site template directory.
func Override(root, relPath string) error {
	if err := ValidateRelPath(relPath); err != nil {
		return err
	}
	root = strings.TrimSpace(root)
	if root == "" {
		return fmt.Errorf("template directory is not configured")
	}

	target := filepath.Join(root, filepath.FromSlash(relPath))
	if _, err := os.Stat(target); err == nil {
		return fmt.Errorf("template already overridden")
	}

	content, err := templateengine.ReadBuiltin(relPath)
	if err != nil {
		return fmt.Errorf("template not found: %s", relPath)
	}
	return Save(root, relPath, []byte(content))
}
