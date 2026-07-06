package updater

import (
	"runtime"
	"strings"
)

// PlatformKeys returns manifest asset lookup keys for the current platform.
func PlatformKeys(name string) []string {
	return []string{
		runtime.GOOS + "_" + runtime.GOARCH,
		runtime.GOOS + "-" + runtime.GOARCH,
		name + "_" + runtime.GOOS + "_" + runtime.GOARCH,
		name + "-" + runtime.GOOS + "-" + runtime.GOARCH,
		name,
	}
}

// SelectAsset picks the best manifest asset for the current platform.
func SelectAsset(name string, manifest Manifest) ManifestAsset {
	for _, key := range PlatformKeys(name) {
		if asset, ok := manifest.Assets[key]; ok && strings.TrimSpace(asset.URL) != "" {
			return asset
		}
	}
	if strings.TrimSpace(manifest.AssetURL) != "" {
		return ManifestAsset{URL: manifest.AssetURL, SHA256: manifest.SHA256}
	}
	return ManifestAsset{}
}

// AssetNameMatches reports whether a release asset filename matches a binary name and platform.
func AssetNameMatches(binaryName, assetName string) bool {
	assetName = strings.TrimSpace(assetName)
	if assetName == binaryName {
		return true
	}
	for _, candidate := range PlatformKeys(binaryName) {
		if assetName == candidate {
			return true
		}
	}
	lower := strings.ToLower(assetName)
	return strings.Contains(lower, strings.ToLower(binaryName)) &&
		strings.Contains(lower, runtime.GOOS) &&
		strings.Contains(lower, runtime.GOARCH)
}
