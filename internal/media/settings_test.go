package media

import "testing"

func TestValidateUploadExtensionAndSize(t *testing.T) {
	cfg := Settings{
		MaxFileSizeBytes: 5 * 1024 * 1024,
		ApprovedExts:     []string{"jpg", "png", "pdf"},
	}
	if err := ValidateUpload(cfg, "photo.jpg", 1024); err != nil {
		t.Fatalf("jpg should be allowed: %v", err)
	}
	if err := ValidateUpload(cfg, "doc.pdf", 1024); err != nil {
		t.Fatalf("pdf should be allowed: %v", err)
	}
	if err := ValidateUpload(cfg, "run.exe", 1024); err == nil {
		t.Fatal("exe should be rejected")
	}
	if err := ValidateUpload(cfg, "big.png", 6*1024*1024); err == nil {
		t.Fatal("oversized file should be rejected")
	}
}

func TestParseApprovedExtensionsWildcard(t *testing.T) {
	exts, allowAll := parseApprovedExtensions("*")
	if !allowAll || len(exts) != 0 {
		t.Fatalf("expected allow all, got %#v allowAll=%v", exts, allowAll)
	}
	exts, allowAll = parseApprovedExtensions("jpg, png;gif webp")
	if allowAll || len(exts) != 4 {
		t.Fatalf("expected 4 extensions, got %#v", exts)
	}
}

func TestApprovedExtensionsLabel(t *testing.T) {
	cfg := Settings{ApprovedExts: []string{"jpg", "png"}}
	if got := cfg.ApprovedExtensionsLabel(); got != "jpg, png" {
		t.Fatalf("label = %q", got)
	}
	cfg.AllowAllTypes = true
	if got := cfg.ApprovedExtensionsLabel(); got != "all file types" {
		t.Fatalf("label = %q", got)
	}
}
