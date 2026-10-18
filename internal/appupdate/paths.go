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
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

// binaryPathCandidates returns paths to check for siblings (raw + symlink target).
func binaryPathCandidates(binaryPath string) []string {
	seen := map[string]struct{}{}
	var out []string
	add := func(p string) {
		p = filepath.Clean(strings.TrimSpace(p))
		if p == "" || p == "." {
			return
		}
		if _, ok := seen[p]; ok {
			return
		}
		seen[p] = struct{}{}
		out = append(out, p)
	}
	add(binaryPath)
	if resolved, err := filepath.EvalSymlinks(binaryPath); err == nil {
		add(resolved)
	}
	return out
}

// findSibling returns the first existing {base}{suffix} beside a candidate binary path.
func findSibling(binaryPath, suffix string) (string, bool) {
	for _, base := range binaryPathCandidates(binaryPath) {
		p := siblingPath(base, suffix)
		if fileExists(p) {
			return p, true
		}
	}
	return siblingPath(binaryPath, suffix), false
}
