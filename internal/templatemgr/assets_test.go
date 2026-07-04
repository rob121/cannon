package templatemgr

import (
	"os"
	"path/filepath"
	"testing"
)

func TestValidateAssetRelPath(t *testing.T) {
	if err := ValidateAssetRelPath("fr/assets/css/custom.css"); err != nil {
		t.Fatalf("valid path: %v", err)
	}
	if err := ValidateAssetRelPath("fr/layout.html"); err == nil {
		t.Fatal("expected html template path to fail")
	}
	if err := ValidateAssetRelPath("fr/assets/logo.png"); err == nil {
		t.Fatal("expected binary extension to fail")
	}
	if err := ValidateAssetRelPath("../secrets.css"); err == nil {
		t.Fatal("expected traversal to fail")
	}
}

func TestSaveAndReadAsset(t *testing.T) {
	root := t.TempDir()
	path := "fr/assets/css/custom.css"
	if err := SaveAsset(root, path, []byte("body { color: red; }")); err != nil {
		t.Fatal(err)
	}
	got, err := ReadAsset(root, path)
	if err != nil {
		t.Fatal(err)
	}
	if got != "body { color: red; }" {
		t.Fatalf("got %q", got)
	}
	if _, err := os.Stat(filepath.Join(root, "fr", "assets", "css", "custom.css")); err != nil {
		t.Fatal(err)
	}
}

func TestEditorModeForAsset(t *testing.T) {
	if EditorModeForAsset("css") != "css" {
		t.Fatal("css mode")
	}
	if EditorModeForAsset(".js") != "javascript" {
		t.Fatal("js mode")
	}
}

func TestListThemeAssets(t *testing.T) {
	root := t.TempDir()
	if err := SaveAsset(root, "fr/assets/site.css", []byte("/* css */")); err != nil {
		t.Fatal(err)
	}
	entries, err := ListThemeAssets(root, "fr")
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].Relative != "site.css" {
		t.Fatalf("entries = %+v", entries)
	}
}

func TestValidateAssetStoragePath(t *testing.T) {
	if err := ValidateAssetStoragePath("fr/assets/logo.png"); err != nil {
		t.Fatalf("png should be allowed for delete: %v", err)
	}
	if err := ValidateAssetStoragePath("fr/assets/css/custom.css"); err != nil {
		t.Fatalf("css should be allowed: %v", err)
	}
	if err := ValidateAssetStoragePath("fr/layout.html"); err == nil {
		t.Fatal("expected template path to fail")
	}
}

func TestDeleteAsset(t *testing.T) {
	root := t.TempDir()
	path := "fr/assets/css/custom.css"
	if err := SaveAsset(root, path, []byte("body { color: red; }")); err != nil {
		t.Fatal(err)
	}
	if err := DeleteAsset(root, path); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(root, "fr", "assets", "css", "custom.css")); !os.IsNotExist(err) {
		t.Fatalf("expected file removed, stat err = %v", err)
	}
	if err := DeleteAsset(root, path); err == nil {
		t.Fatal("expected error deleting missing file")
	}
}
