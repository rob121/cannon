package media

import (
	"image"
	"image/gif"
	"image/jpeg"
	"image/png"
	"os"
	"path/filepath"
	"strings"
)

const thumbMaxWidth = 320

// ThumbnailPath returns the filesystem path for a generated thumbnail next to the original.
func ThumbnailPath(originalPath string) string {
	ext := filepath.Ext(originalPath)
	base := strings.TrimSuffix(originalPath, ext)
	return base + "_thumb" + ext
}

// WebThumbnailPath returns the public URL for a thumbnail derived from an asset path.
func WebThumbnailPath(webPath string) string {
	ext := filepath.Ext(webPath)
	base := strings.TrimSuffix(webPath, ext)
	return base + "_thumb" + ext
}

// ResolveWebThumbnail returns the thumbnail URL when a generated thumb exists, otherwise webPath.
func ResolveWebThumbnail(assetsDir, webPath string) string {
	if webPath == "" || assetsDir == "" {
		return webPath
	}
	thumbWeb := WebThumbnailPath(webPath)
	thumbLocal := filepath.Join(assetsDir, strings.TrimPrefix(thumbWeb, "/assets/"))
	if _, err := os.Stat(thumbLocal); err == nil {
		return thumbWeb
	}
	return webPath
}

// GenerateThumbnail creates a resized copy of an image if it is wider than thumbMaxWidth.
func GenerateThumbnail(srcPath string) (string, error) {
	f, err := os.Open(srcPath)
	if err != nil {
		return "", err
	}
	defer f.Close()
	img, _, err := image.Decode(f)
	if err != nil {
		return "", err
	}
	bounds := img.Bounds()
	w := bounds.Dx()
	if w <= thumbMaxWidth {
		return "", nil
	}
	h := bounds.Dy()
	newH := h * thumbMaxWidth / w
	if newH < 1 {
		newH = 1
	}
	dst := image.NewRGBA(image.Rect(0, 0, thumbMaxWidth, newH))
	for y := 0; y < newH; y++ {
		srcY := bounds.Min.Y + y*h/newH
		for x := 0; x < thumbMaxWidth; x++ {
			srcX := bounds.Min.X + x*w/thumbMaxWidth
			dst.Set(x, y, img.At(srcX, srcY))
		}
	}
	outPath := ThumbnailPath(srcPath)
	out, err := os.Create(outPath)
	if err != nil {
		return "", err
	}
	defer out.Close()
	ext := strings.ToLower(filepath.Ext(srcPath))
	switch ext {
	case ".png":
		err = png.Encode(out, dst)
	case ".gif":
		err = gif.Encode(out, dst, nil)
	default:
		err = jpeg.Encode(out, dst, &jpeg.Options{Quality: 85})
	}
	if err != nil {
		_ = os.Remove(outPath)
		return "", err
	}
	return outPath, nil
}
