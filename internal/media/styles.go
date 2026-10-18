package media

import (
	"fmt"
	"image"
	"image/gif"
	"image/jpeg"
	"image/png"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/disintegration/imaging"
	_ "golang.org/x/image/webp"
)

const (
	ModeFit           = "fit"
	StyleKeyThumbnail = "thumbnail"
	thumbSuffix       = "thumb"
	jpegQuality       = 85
)

var styleKeyPattern = regexp.MustCompile(`^[a-z0-9_]+$`)

// Style describes a named image derivative (global media setting).
type Style struct {
	Key    string `json:"key"`
	Width  int    `json:"width"`
	Height int    `json:"height"` // 0 = proportional from width
	Mode   string `json:"mode"`   // fit (v1)
}

// DefaultStyles returns the seeded global image styles.
func DefaultStyles() []Style {
	return []Style{
		{Key: StyleKeyThumbnail, Width: 320, Mode: ModeFit},
		{Key: "medium", Width: 640, Mode: ModeFit},
		{Key: "large", Width: 1200, Mode: ModeFit},
	}
}

// StyleSuffix returns the filename suffix for a style key (without leading underscore).
// thumbnail maps to "thumb" for backward compatibility with existing _thumb files.
func StyleSuffix(styleKey string) string {
	key := normalizeStyleKey(styleKey)
	if key == StyleKeyThumbnail || key == thumbSuffix {
		return thumbSuffix
	}
	return key
}

// DerivativePath returns the filesystem path for a style derivative next to the original.
func DerivativePath(originalPath, styleKey string) string {
	ext := filepath.Ext(originalPath)
	base := strings.TrimSuffix(originalPath, ext)
	return base + "_" + StyleSuffix(styleKey) + ext
}

// WebDerivativePath returns the public URL for a style derivative.
func WebDerivativePath(webPath, styleKey string) string {
	ext := filepath.Ext(webPath)
	base := strings.TrimSuffix(webPath, ext)
	return base + "_" + StyleSuffix(styleKey) + ext
}

// ResolveWebStyle returns the derivative URL when the file exists, otherwise webPath.
func ResolveWebStyle(assetsDir, webPath, styleKey string) string {
	if webPath == "" || assetsDir == "" || strings.TrimSpace(styleKey) == "" {
		return webPath
	}
	derivWeb := WebDerivativePath(webPath, styleKey)
	local := filepath.Join(assetsDir, strings.TrimPrefix(derivWeb, "/assets/"))
	if _, err := os.Stat(local); err == nil {
		return derivWeb
	}
	return webPath
}

// GenerateStyles decodes srcPath once and writes a derivative for each style.
// Non-raster images (e.g. SVG) and decode failures return an error; individual
// style write failures are collected and returned as a multi-error string.
func GenerateStyles(srcPath string, styles []Style) error {
	if len(styles) == 0 {
		styles = DefaultStyles()
	}
	ext := strings.ToLower(filepath.Ext(srcPath))
	if ext == ".svg" {
		return fmt.Errorf("image styles are not generated for SVG")
	}
	f, err := os.Open(srcPath)
	if err != nil {
		return err
	}
	defer f.Close()
	img, _, err := image.Decode(f)
	if err != nil {
		return err
	}
	var errs []string
	for _, style := range styles {
		if err := writeStyleDerivative(srcPath, img, style); err != nil {
			errs = append(errs, fmt.Sprintf("%s: %v", style.Key, err))
		}
	}
	if len(errs) > 0 {
		return fmt.Errorf("generate styles: %s", strings.Join(errs, "; "))
	}
	return nil
}

// RemoveDerivatives deletes known style sibling files for an original asset path.
func RemoveDerivatives(originalPath string, styles []Style) {
	if originalPath == "" {
		return
	}
	if len(styles) == 0 {
		styles = DefaultStyles()
	}
	seen := map[string]struct{}{}
	for _, style := range styles {
		path := DerivativePath(originalPath, style.Key)
		if _, ok := seen[path]; ok {
			continue
		}
		seen[path] = struct{}{}
		_ = os.Remove(path)
	}
	// Always attempt legacy _thumb in case styles omit thumbnail.
	_ = os.Remove(DerivativePath(originalPath, StyleKeyThumbnail))
}

func writeStyleDerivative(srcPath string, img image.Image, style Style) error {
	style, err := normalizeStyle(style)
	if err != nil {
		return err
	}
	bounds := img.Bounds()
	srcW := bounds.Dx()
	srcH := bounds.Dy()
	if srcW < 1 || srcH < 1 {
		return fmt.Errorf("invalid image dimensions")
	}

	dstW, dstH := fitDimensions(srcW, srcH, style.Width, style.Height)
	if dstW == srcW && dstH == srcH {
		// Already within bounds; skip derivative (ResolveWebStyle falls back to original).
		return nil
	}

	dst := imaging.Resize(img, dstW, dstH, imaging.CatmullRom)
	outPath := DerivativePath(srcPath, style.Key)
	out, err := os.Create(outPath)
	if err != nil {
		return err
	}
	defer out.Close()

	ext := strings.ToLower(filepath.Ext(srcPath))
	switch ext {
	case ".png":
		err = png.Encode(out, dst)
	case ".gif":
		err = gif.Encode(out, dst, nil)
	default:
		// jpeg/jpg/webp (decode only for webp) and unknown raster → JPEG
		err = jpeg.Encode(out, dst, &jpeg.Options{Quality: jpegQuality})
	}
	if err != nil {
		_ = os.Remove(outPath)
		return err
	}
	return nil
}

// fitDimensions returns target size for ModeFit: max width (and optional max height), never upscaling.
func fitDimensions(srcW, srcH, maxW, maxH int) (int, int) {
	if maxW < 1 {
		maxW = srcW
	}
	dstW, dstH := srcW, srcH
	if srcW > maxW {
		dstW = maxW
		dstH = srcH * maxW / srcW
		if dstH < 1 {
			dstH = 1
		}
	}
	if maxH > 0 && dstH > maxH {
		dstH = maxH
		dstW = srcW * maxH / srcH
		if dstW < 1 {
			dstW = 1
		}
		// Re-clamp width if height clamp made width exceed maxW.
		if dstW > maxW {
			dstW = maxW
			dstH = srcH * maxW / srcW
			if dstH < 1 {
				dstH = 1
			}
		}
	}
	return dstW, dstH
}

func normalizeStyleKey(key string) string {
	return strings.ToLower(strings.TrimSpace(key))
}

func normalizeStyle(s Style) (Style, error) {
	s.Key = normalizeStyleKey(s.Key)
	if s.Key == "" || !styleKeyPattern.MatchString(s.Key) {
		return Style{}, fmt.Errorf("invalid style key %q", s.Key)
	}
	if s.Width < 1 || s.Width > 4096 {
		return Style{}, fmt.Errorf("style %q width must be 1–4096", s.Key)
	}
	if s.Height < 0 || s.Height > 4096 {
		return Style{}, fmt.Errorf("style %q height must be 0–4096", s.Key)
	}
	mode := strings.ToLower(strings.TrimSpace(s.Mode))
	if mode == "" {
		mode = ModeFit
	}
	if mode != ModeFit {
		return Style{}, fmt.Errorf("style %q mode %q is not supported", s.Key, s.Mode)
	}
	s.Mode = mode
	return s, nil
}

// ValidateStyles sanitizes and dedupes a style list. Empty input yields defaults.
func ValidateStyles(in []Style) ([]Style, error) {
	if len(in) == 0 {
		return DefaultStyles(), nil
	}
	out := make([]Style, 0, len(in))
	seen := map[string]struct{}{}
	for _, s := range in {
		ns, err := normalizeStyle(s)
		if err != nil {
			return nil, err
		}
		if _, ok := seen[ns.Key]; ok {
			continue
		}
		seen[ns.Key] = struct{}{}
		out = append(out, ns)
	}
	if len(out) == 0 {
		return DefaultStyles(), nil
	}
	return out, nil
}
