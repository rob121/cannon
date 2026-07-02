package extensions

import (
	"crypto/sha256"
	"encoding/hex"
	"path/filepath"
	"strings"

	"github.com/rob121/cannon/extension"
)

// RouteHash returns the stable public route id for an extension on a site.
func RouteHash(extensionName, siteID string) string {
	return extension.RouteHash(extensionName, siteID)
}

// RouteHashFromSocket extracts the route hash from an extension socket path.
func RouteHashFromSocket(socketPath string) string {
	base := filepath.Base(socketPath)
	return strings.TrimSuffix(base, ".sock")
}

// PublicDataURL builds /ext/{route_hash}/{relativePath}.
func PublicDataURL(extensionName, siteID, relativePath string) string {
	return extension.PublicDataURL(extensionName, siteID, relativePath)
}

func routeHashBytes(extensionName, siteID string) [32]byte {
	return sha256.Sum256([]byte(strings.TrimSpace(extensionName) + ":" + strings.TrimSpace(siteID)))
}

func routeHashHex(extensionName, siteID string) string {
	sum := routeHashBytes(extensionName, siteID)
	return hex.EncodeToString(sum[:16])
}
