package extension

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
)

// RouteHash returns the stable public route id for an extension on a site.
// Cannon uses this value in /ext/{route_hash}/... URLs.
func RouteHash(extensionName, siteID string) string {
	sum := sha256.Sum256([]byte(strings.TrimSpace(extensionName) + ":" + strings.TrimSpace(siteID)))
	return hex.EncodeToString(sum[:16])
}

// PublicDataURL builds the public proxy URL for a data route.
// relativePath is the handler path registered with HandleData, for example "contact/submit".
func PublicDataURL(extensionName, siteID, relativePath string) string {
	rel := strings.Trim(relativePath, "/")
	if rel == "" {
		return "/ext/" + RouteHash(extensionName, siteID)
	}
	return "/ext/" + RouteHash(extensionName, siteID) + "/" + rel
}

// DataPath returns the data route path from a wire request.
func DataPath(req WireRequest) string {
	if p := strings.TrimSpace(req.DataPath); p != "" {
		return strings.Trim(p, "/")
	}
	const prefix = "/ext/"
	if strings.HasPrefix(req.URL, prefix) {
		parts := strings.Split(strings.TrimPrefix(req.URL, prefix), "/")
		if len(parts) > 1 {
			return strings.Join(parts[1:], "/")
		}
	}
	return ""
}
