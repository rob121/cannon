package appupdate

import (
	"os"
	"path/filepath"
	"strings"
)

// resolveBinaryPath returns the running executable path (or override if set),
// normalized to the live install name (strips .previous / .latest suffixes).
func resolveBinaryPath(override string) (string, error) {
	if v := strings.TrimSpace(override); v != "" {
		return normalizeLiveBinaryPath(v), nil
	}
	path, err := os.Executable()
	if err != nil {
		return "", err
	}
	return normalizeLiveBinaryPath(path), nil
}

// normalizeLiveBinaryPath maps a path that may be the renamed running inode
// (e.g. cannon.previous after a staged swap) back to the canonical live name
// (cannon). Without this, staging writes cannon.previous.latest.
func normalizeLiveBinaryPath(path string) string {
	path = filepath.Clean(strings.TrimSpace(path))
	if path == "" || path == "." {
		return path
	}
	dir := filepath.Dir(path)
	base := filepath.Base(path)
	for {
		switch {
		case strings.HasSuffix(base, ".previous"):
			base = strings.TrimSuffix(base, ".previous")
		case strings.HasSuffix(base, ".latest"):
			base = strings.TrimSuffix(base, ".latest")
		case strings.HasSuffix(base, ".rollback-tmp"):
			base = strings.TrimSuffix(base, ".rollback-tmp")
		default:
			return filepath.Join(dir, base)
		}
	}
}

func siblingPath(binaryPath, suffix string) string {
	live := normalizeLiveBinaryPath(binaryPath)
	dir := filepath.Dir(live)
	base := filepath.Base(live)
	return filepath.Join(dir, base+suffix)
}

func latestPath(binaryPath string) string {
	return siblingPath(binaryPath, ".latest")
}

func previousPath(binaryPath string) string {
	return siblingPath(binaryPath, ".previous")
}

// legacyMistakenLatest is the bad name written when Executable pointed at *.previous.
func legacyMistakenLatest(binaryPath string) string {
	live := normalizeLiveBinaryPath(binaryPath)
	return filepath.Join(filepath.Dir(live), filepath.Base(live)+".previous.latest")
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

// binaryPathCandidates returns paths to check for siblings (normalized + symlink target).
func binaryPathCandidates(binaryPath string) []string {
	seen := map[string]struct{}{}
	var out []string
	add := func(p string) {
		p = normalizeLiveBinaryPath(p)
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
	if resolved, err := filepath.EvalSymlinks(normalizeLiveBinaryPath(binaryPath)); err == nil {
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
	if suffix == ".latest" {
		if mistaken := legacyMistakenLatest(binaryPath); fileExists(mistaken) {
			return mistaken, true
		}
	}
	return siblingPath(binaryPath, suffix), false
}
