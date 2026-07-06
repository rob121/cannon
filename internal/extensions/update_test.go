package extensions

import (
	"net/http"
	"net/http/httptest"
	"runtime"
	"testing"
	"time"

	"github.com/rob121/cannon/internal/models"
)

func TestNewerVersion(t *testing.T) {
	if !newerVersion("v0.2.0", "0.1.0") {
		t.Fatal("expected v0.2.0 to be newer than 0.1.0")
	}
	if newerVersion("v0.1.0", "0.1.0") {
		t.Fatal("expected equivalent versions to not be newer")
	}
}

func TestGitHubLatestAPIURL(t *testing.T) {
	got := githubLatestAPIURL("https://github.com/rob121/cannon-ext-gcalendar/releases/download")
	want := "https://api.github.com/repos/rob121/cannon-ext-gcalendar/releases/latest"
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
	if got := githubLatestAPIURL("https://example.com/releases/download"); got != "" {
		t.Fatalf("expected non-GitHub URL to be ignored, got %q", got)
	}
}

func TestUpdateManifestURLForGitHubReleaseBase(t *testing.T) {
	got := updateManifestURL("https://github.com/rob121/cannon-ext-gzip/releases/download")
	want := "https://github.com/rob121/cannon-ext-gzip/releases/latest/download/cannon-extension.json"
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestFetchUpdateManifest(t *testing.T) {
	platform := runtime.GOOS + "_" + runtime.GOARCH
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/latest/download/cannon-extension.json" {
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
		_, _ = w.Write([]byte(`{
			"name": "demo-extension",
			"version": "0.2.0",
			"assets": {
				"` + platform + `": {
					"url": "` + "http://example.test/demo-extension" + `",
					"sha256": "sha256:abc123"
				}
			}
		}`))
	}))
	defer server.Close()

	m := NewManager(nil, nil)
	m.updateClient = server.Client()
	info, err := m.fetchUpdateManifest(models.Extension{Name: "demo-extension"}, server.URL)
	if err != nil {
		t.Fatal(err)
	}
	if info.Version != "0.2.0" || info.URL != "http://example.test/demo-extension" || info.SHA256 != "abc123" {
		t.Fatalf("unexpected update info: %+v", info)
	}
}

func TestFetchGitHubLatestSelectsPlatformAsset(t *testing.T) {
	assetName := "demo-extension-" + runtime.GOOS + "-" + runtime.GOARCH
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{
			"tag_name": "v0.3.0",
			"assets": [
				{"name": "demo-extension-linux-amd64", "browser_download_url": "http://example.test/linux"},
				{"name": "` + assetName + `", "browser_download_url": "http://example.test/platform", "digest": "sha256:def456"}
			]
		}`))
	}))
	defer server.Close()

	m := NewManager(nil, nil)
	m.updateClient = &http.Client{Timeout: time.Second, Transport: rewriteHostTransport{base: server.URL}}
	info, err := m.fetchGitHubLatest(models.Extension{Name: "demo-extension", UpdateURLBase: "https://github.com/rob121/demo-extension/releases/download"}, "https://api.github.com/repos/rob121/demo-extension/releases/latest")
	if err != nil {
		t.Fatal(err)
	}
	if info.Version != "v0.3.0" || info.URL != "http://example.test/platform" || info.SHA256 != "def456" {
		t.Fatalf("unexpected update info: %+v", info)
	}
}

type rewriteHostTransport struct {
	base string
}

func (rt rewriteHostTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	rewritten, err := http.NewRequest(req.Method, rt.base, req.Body)
	if err != nil {
		return nil, err
	}
	rewritten.Header = req.Header.Clone()
	return http.DefaultTransport.RoundTrip(rewritten)
}
