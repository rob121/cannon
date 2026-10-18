package media

import (
	"image"
	"image/color"
	"image/jpeg"
	"os"
	"path/filepath"
	"testing"
)

func TestStyleSuffixThumbnailCompat(t *testing.T) {
	if got := StyleSuffix("thumbnail"); got != "thumb" {
		t.Fatalf("StyleSuffix(thumbnail)=%q", got)
	}
	if got := StyleSuffix("medium"); got != "medium" {
		t.Fatalf("StyleSuffix(medium)=%q", got)
	}
}

func TestDerivativePathNaming(t *testing.T) {
	got := DerivativePath("/assets/media/a/photo.jpg", "thumbnail")
	if got != "/assets/media/a/photo_thumb.jpg" {
		t.Fatalf("thumbnail path=%q", got)
	}
	got = WebDerivativePath("/assets/media/a/photo.jpg", "medium")
	if got != "/assets/media/a/photo_medium.jpg" {
		t.Fatalf("medium web path=%q", got)
	}
}

func TestResolveWebStyleFallback(t *testing.T) {
	dir := t.TempDir()
	web := "/assets/media/missing.jpg"
	if got := ResolveWebStyle(dir, web, "medium"); got != web {
		t.Fatalf("expected original, got %q", got)
	}
}

func TestGenerateStylesNoUpscale(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "small.jpg")
	writeTestJPEG(t, src, 100, 80)

	styles := []Style{{Key: "medium", Width: 640, Mode: ModeFit}}
	if err := GenerateStyles(src, styles); err != nil {
		t.Fatal(err)
	}
	deriv := DerivativePath(src, "medium")
	if _, err := os.Stat(deriv); !os.IsNotExist(err) {
		t.Fatalf("expected no derivative when source smaller than style, err=%v", err)
	}
}

func TestGenerateStylesResizes(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "wide.jpg")
	writeTestJPEG(t, src, 800, 400)

	styles := []Style{
		{Key: "thumbnail", Width: 320, Mode: ModeFit},
		{Key: "medium", Width: 640, Mode: ModeFit},
	}
	if err := GenerateStyles(src, styles); err != nil {
		t.Fatal(err)
	}
	thumb := DerivativePath(src, "thumbnail")
	med := DerivativePath(src, "medium")
	if _, err := os.Stat(thumb); err != nil {
		t.Fatalf("thumb missing: %v", err)
	}
	if _, err := os.Stat(med); err != nil {
		t.Fatalf("medium missing: %v", err)
	}
	assertJPEGWidth(t, thumb, 320)
	assertJPEGWidth(t, med, 640)

	assetsDir := dir
	// Place under assets-style layout for resolve
	web := "/assets/wide.jpg"
	// Resolve uses assetsDir + trim /assets/
	// Put files where resolve expects: assetsDir/wide_thumb.jpg
	if got := ResolveWebStyle(dir, web, "thumbnail"); got != "/assets/wide_thumb.jpg" {
		t.Fatalf("resolve thumb=%q", got)
	}
	_ = assetsDir
}

func TestValidateStyles(t *testing.T) {
	got, err := ValidateStyles([]Style{
		{Key: "Teaser", Width: 400, Mode: "fit"},
		{Key: "teaser", Width: 500, Mode: "fit"}, // dedupe
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].Key != "teaser" || got[0].Width != 400 {
		t.Fatalf("got %#v", got)
	}
	if _, err := ValidateStyles([]Style{{Key: "Bad Key!", Width: 100}}); err == nil {
		t.Fatal("expected invalid key error")
	}
}

func TestFitDimensions(t *testing.T) {
	w, h := fitDimensions(1000, 500, 320, 0)
	if w != 320 || h != 160 {
		t.Fatalf("got %dx%d", w, h)
	}
	w, h = fitDimensions(100, 80, 320, 0)
	if w != 100 || h != 80 {
		t.Fatalf("no upscale: got %dx%d", w, h)
	}
}

func writeTestJPEG(t *testing.T, path string, w, h int) {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, color.RGBA{R: uint8(x), G: uint8(y), B: 40, A: 255})
		}
	}
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if err := jpeg.Encode(f, img, &jpeg.Options{Quality: 90}); err != nil {
		t.Fatal(err)
	}
}

func assertJPEGWidth(t *testing.T, path string, want int) {
	t.Helper()
	f, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	cfg, err := jpeg.DecodeConfig(f)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Width != want {
		t.Fatalf("%s width=%d want %d", path, cfg.Width, want)
	}
}
