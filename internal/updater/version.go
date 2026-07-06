package updater

import (
	"strings"

	"github.com/Masterminds/semver/v3"
)

// NewerVersion reports whether latest is newer than current using semver when possible.
func NewerVersion(latest, current string) bool {
	latest = strings.TrimSpace(latest)
	current = strings.TrimSpace(current)
	if latest == "" {
		return false
	}
	if current == "" || current == "dev" {
		return true
	}
	lv, lerr := semver.NewVersion(strings.TrimPrefix(latest, "v"))
	cv, cerr := semver.NewVersion(strings.TrimPrefix(current, "v"))
	if lerr == nil && cerr == nil {
		return lv.GreaterThan(cv)
	}
	return strings.TrimPrefix(latest, "v") != strings.TrimPrefix(current, "v")
}

// FirstNonEmpty returns the first non-blank string.
func FirstNonEmpty(values ...string) string {
	for _, value := range values {
		if v := strings.TrimSpace(value); v != "" {
			return v
		}
	}
	return ""
}

// NormalizeDigest strips an optional sha256: prefix.
func NormalizeDigest(value string) string {
	value = strings.TrimSpace(value)
	value = strings.TrimPrefix(value, "sha256:")
	return strings.ToLower(value)
}
