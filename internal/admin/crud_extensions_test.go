package admin

import (
	"runtime"
	"testing"

	"github.com/rob121/cannon/internal/extensions"
	"github.com/rob121/cannon/internal/models"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestExtensionDisplayTitle(t *testing.T) {
	if got := extensionDisplayTitle("Contact Forms", "", "cannon-extension-contact"); got != "Contact Forms" {
		t.Fatalf("got %q", got)
	}
	if got := extensionDisplayTitle("", "Menu Label", "binary"); got != "Menu Label" {
		t.Fatalf("got %q", got)
	}
	if got := extensionDisplayTitle("", "", "binary"); got != "binary" {
		t.Fatalf("got %q", got)
	}
}

func TestMergeExtensionMetaUsesCachedRow(t *testing.T) {
	meta := mergeExtensionMeta(extensions.MetaSummary{Available: false}, models.Extension{
		Name:          "demo",
		Title:         "Demo",
		Description:   "Cached description",
		Version:       "1.0.0",
		UpdateURLBase: "https://github.com/rob121/demo/releases/download",
	})
	if !meta.Available || meta.Description != "Cached description" || meta.Title != "Demo" || meta.UpdateURLBase == "" {
		t.Fatalf("meta = %+v", meta)
	}
}

func TestExtensionReorder(t *testing.T) {
	db := testAdminDB(t)
	rows := []models.Extension{
		{ExtensionID: 1, Name: "a", Sort: 0, Socket: "/tmp/a.sock", Status: models.StatusInactive},
		{ExtensionID: 2, Name: "b", Sort: 1, Socket: "/tmp/b.sock", Status: models.StatusInactive},
		{ExtensionID: 3, Name: "c", Sort: 2, Socket: "/tmp/c.sock", Status: models.StatusInactive},
	}
	for _, row := range rows {
		if err := db.Create(&row).Error; err != nil {
			t.Fatal(err)
		}
	}
	if err := extensionReorder(db, 2, 1); err != nil {
		t.Fatal(err)
	}
	var ordered []models.Extension
	if err := db.Order("sort asc, extension_id asc").Find(&ordered).Error; err != nil {
		t.Fatal(err)
	}
	if len(ordered) != 3 || ordered[0].Name != "a" || ordered[1].Name != "c" || ordered[2].Name != "b" {
		t.Fatalf("ordered = %#v", ordered)
	}
}

func testAdminDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&models.Extension{}); err != nil {
		t.Fatal(err)
	}
	return db
}

func TestResolveExtensionInstallNameFromGitHubRelease(t *testing.T) {
	releaseURL := "https://github.com/rob121/cannon-ext-captcha-cfturnstile/releases/tag/v0.1.0"
	assetURL := "https://github.com/rob121/cannon-ext-captcha-cfturnstile/releases/download/v0.1.0/cannon-ext-captcha-cfturnstile-darwin_arm64"
	got, err := resolveExtensionInstallName("", releaseURL, "cannon-ext-captcha-cfturnstile", assetURL)
	if err != nil {
		t.Fatal(err)
	}
	if got != "cannon-ext-captcha-cfturnstile" {
		t.Fatalf("name = %q", got)
	}
}

func TestResolveExtensionInstallNameSkipsInvalidOverride(t *testing.T) {
	releaseURL := "https://github.com/rob121/cannon-ext-captcha-cfturnstile/releases/tag/v0.1.0"
	got, err := resolveExtensionInstallName(releaseURL, releaseURL, "cannon-ext-captcha-cfturnstile", "")
	if err != nil {
		t.Fatal(err)
	}
	if got != "cannon-ext-captcha-cfturnstile" {
		t.Fatalf("name = %q, want manifest name fallback", got)
	}
}

func TestSanitizedNameOverrideFromReleaseURL(t *testing.T) {
	if got := sanitizedNameOverride("https://github.com/rob121/cannon-ext-captcha-cfturnstile/releases/tag/v0.1.0"); got != "cannon-ext-captcha-cfturnstile" {
		t.Fatalf("got %q", got)
	}
}

func TestResolveExtensionInstallNameRejectsInvalidCandidates(t *testing.T) {
	if _, err := resolveExtensionInstallName("bad name", "", "", ""); err == nil {
		t.Fatal("expected invalid name to fail when no fallback exists")
	}
}

func TestGithubRepoName(t *testing.T) {
	if got := githubRepoName("https://github.com/rob121/cannon-ext-captcha-cfturnstile/releases/tag/v0.1.0"); got != "cannon-ext-captcha-cfturnstile" {
		t.Fatalf("repo = %q", got)
	}
}

func TestStripPlatformSuffix(t *testing.T) {
	if got := stripPlatformSuffix("cannon-ext-captcha-cfturnstile-darwin_arm64"); got != "cannon-ext-captcha-cfturnstile" {
		t.Fatalf("got %q", got)
	}
}

func TestExtensionManifestURLFromGitHubRelease(t *testing.T) {
	manifestURL, updateBase, ok := extensionManifestURL("https://github.com/rob121/cannon-ext-gzip/releases/tag/v0.1.1")
	if !ok {
		t.Fatal("expected GitHub release URL to resolve to a manifest")
	}
	if want := "https://github.com/rob121/cannon-ext-gzip/releases/download/v0.1.1/cannon-extension.json"; manifestURL != want {
		t.Fatalf("manifest URL = %q, want %q", manifestURL, want)
	}
	if want := "https://github.com/rob121/cannon-ext-gzip/releases/download"; updateBase != want {
		t.Fatalf("update base = %q, want %q", updateBase, want)
	}
}

func TestExtensionManifestURLFromGitHubRepository(t *testing.T) {
	manifestURL, updateBase, ok := extensionManifestURL("https://github.com/rob121/cannon-ext-gzip")
	if !ok {
		t.Fatal("expected GitHub repository URL to resolve to latest manifest")
	}
	if want := "https://github.com/rob121/cannon-ext-gzip/releases/latest/download/cannon-extension.json"; manifestURL != want {
		t.Fatalf("manifest URL = %q, want %q", manifestURL, want)
	}
	if want := "https://github.com/rob121/cannon-ext-gzip/releases/download"; updateBase != want {
		t.Fatalf("update base = %q, want %q", updateBase, want)
	}
}

func TestGithubUpdateBaseFromReleaseAsset(t *testing.T) {
	got := githubUpdateBase("https://github.com/rob121/cannon-ext-gzip/releases/download/v0.1.1/cannon-ext-gzip-darwin_arm64")
	if want := "https://github.com/rob121/cannon-ext-gzip/releases/download"; got != want {
		t.Fatalf("update base = %q, want %q", got, want)
	}
}

func TestSelectExtensionManifestAssetUsesCurrentPlatform(t *testing.T) {
	key := runtime.GOOS + "_" + runtime.GOARCH
	asset := selectExtensionManifestAsset(extensionDownloadManifest{
		Name: "cannon-ext-demo",
		Assets: map[string]extensionManifestAsset{
			"linux_amd64": {URL: "https://example.test/linux"},
			key:           {URL: "https://example.test/current", SHA256: "abc123"},
		},
	})
	if asset.URL != "https://example.test/current" || asset.SHA256 != "abc123" {
		t.Fatalf("asset = %+v", asset)
	}
}
