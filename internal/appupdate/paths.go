package appupdate

import (
	"os"
	"path/filepath"
	"strings"
)

// resolveBinaryPath returns the running executable path (or override if set).
func resolveBinaryPath(override string) (string, error) {
	if v := strings.TrimSpace(override); v != "" {
		return filepath.Clean(v), nil
	}
	path, err := os.Executable()
	if err != nil {
		return "", err
	}
	return filepath.Clean(path), nil
}

func siblingPath(binaryPath, suffix string) string {
	dir := filepath.Dir(binaryPath)
	base := filepath.Base(binaryPath)
	return filepath.Join(dir, base+suffix)
}

func latestPath(binaryPath string) string {
	return siblingPath(binaryPath, ".latest")
}

func previousPath(binaryPath string) string {
	return siblingPath(binaryPath, ".previous")
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
