package media

import "os"

// ThumbnailPath returns the filesystem path for the thumbnail style derivative.
func ThumbnailPath(originalPath string) string {
	return DerivativePath(originalPath, StyleKeyThumbnail)
}

// WebThumbnailPath returns the public URL for the thumbnail style derivative.
func WebThumbnailPath(webPath string) string {
	return WebDerivativePath(webPath, StyleKeyThumbnail)
}

// ResolveWebThumbnail returns the thumbnail URL when a generated thumb exists, otherwise webPath.
func ResolveWebThumbnail(assetsDir, webPath string) string {
	return ResolveWebStyle(assetsDir, webPath, StyleKeyThumbnail)
}

// GenerateThumbnail creates the thumbnail style derivative (and only that style).
// Prefer GenerateStyles with LoadStyles for full style sets on upload.
func GenerateThumbnail(srcPath string) (string, error) {
	styles := []Style{{Key: StyleKeyThumbnail, Width: 320, Mode: ModeFit}}
	if err := GenerateStyles(srcPath, styles); err != nil {
		return "", err
	}
	out := ThumbnailPath(srcPath)
	if _, err := os.Stat(out); err != nil {
		return "", nil
	}
	return out, nil
}
