package updater

import (
	"net/url"
	"path"
	"strings"
)

// Manifest describes a release artifact set.
type Manifest struct {
	Name          string                    `json:"name"`
	Version       string                    `json:"version"`
	LatestVersion string                    `json:"latest_version"`
	TagName       string                    `json:"tag_name"`
	AssetURL      string                    `json:"asset_url"`
	SHA256        string                    `json:"sha256"`
	Assets        map[string]ManifestAsset  `json:"assets"`
}

// ManifestAsset is a platform-specific release binary.
type ManifestAsset struct {
	URL    string `json:"url"`
	SHA256 string `json:"sha256"`
}

// Info is the resolved update target for the current platform.
type Info struct {
	Version string
	URL     string
	SHA256  string
}

// ManifestURL returns the latest manifest URL for a release base.
func ManifestURL(rawBase, manifestName string) string {
	base := strings.TrimRight(strings.TrimSpace(rawBase), "/")
	manifestName = strings.TrimSpace(manifestName)
	if manifestName == "" {
		manifestName = "manifest.json"
	}
	u, err := url.Parse(base)
	if err == nil && strings.EqualFold(u.Host, "github.com") {
		parts := strings.Split(strings.Trim(u.Path, "/"), "/")
		if len(parts) >= 4 && parts[2] == "releases" && parts[3] == "download" {
			return "https://github.com/" + path.Join(parts[0], parts[1], "releases", "latest", "download", manifestName)
		}
	}
	return base + "/latest/download/" + manifestName
}

// GitHubLatestAPIURL derives the GitHub latest-release API URL from a releases/download base.
func GitHubLatestAPIURL(rawBase string) string {
	u, err := url.Parse(strings.TrimSpace(rawBase))
	if err != nil || !strings.EqualFold(u.Host, "github.com") {
		return ""
	}
	parts := strings.Split(strings.Trim(u.Path, "/"), "/")
	if len(parts) < 4 || parts[2] != "releases" || parts[3] != "download" {
		return ""
	}
	return "https://api.github.com/repos/" + path.Join(parts[0], parts[1], "releases", "latest")
}

// DefaultAssetURL builds a conventional GitHub release asset URL.
func DefaultAssetURL(base, version, name string) string {
	base = strings.TrimRight(strings.TrimSpace(base), "/")
	version = strings.TrimSpace(version)
	name = strings.TrimSpace(name)
	if base == "" || version == "" || name == "" {
		return ""
	}
	return base + "/" + version + "/" + name
}
