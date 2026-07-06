package updater

import (
	"net/http"
	"net/http/httptest"
	"runtime"
	"testing"
	"time"
)

func TestNewerVersion(t *testing.T) {
	if !NewerVersion("v0.2.0", "0.1.0") {
		t.Fatal("expected v0.2.0 to be newer than 0.1.0")
	}
	if NewerVersion("v0.1.0", "0.1.0") {
		t.Fatal("expected equivalent versions to not be newer")
	}
	if !NewerVersion("v0.1.0", "dev") {
		t.Fatal("expected any release to be newer than dev")
	}
}

func TestManifestURLForGitHubReleaseBase(t *testing.T) {
	got := ManifestURL("https://github.com/rob121/cannon/releases/download", "cannon.json")
	want := "https://github.com/rob121/cannon/releases/latest/download/cannon.json"
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestClientFetchManifest(t *testing.T) {
	platform := runtime.GOOS + "_" + runtime.GOARCH
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/latest/download/cannon.json" {
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
		_, _ = w.Write([]byte(`{
			"name": "cannon",
			"version": "v0.2.0",
			"assets": {
				"` + platform + `": {
					"url": "http://example.test/cannon",
					"sha256": "sha256:abc123"
				}
			}
		}`))
	}))
	defer server.Close()

	client := &Client{HTTP: server.Client(), Manifest: "cannon.json"}
	info, err := client.LatestInfo(server.URL, "cannon")
	if err != nil {
		t.Fatal(err)
	}
	if info.Version != "v0.2.0" || info.URL != "http://example.test/cannon" || info.SHA256 != "abc123" {
		t.Fatalf("unexpected update info: %+v", info)
	}
}

func TestClientFetchGitHubLatest(t *testing.T) {
	assetName := "cannon-" + runtime.GOOS + "_" + runtime.GOARCH
	decoyName := "cannon-darwin_arm64"
	if runtime.GOOS == "darwin" && runtime.GOARCH == "arm64" {
		decoyName = "cannon-linux_amd64"
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{
			"tag_name": "v0.3.0",
			"assets": [
				{"name": "` + decoyName + `", "browser_download_url": "http://example.test/decoy"},
				{"name": "` + assetName + `", "browser_download_url": "http://example.test/platform", "digest": "sha256:def456"}
			]
		}`))
	}))
	defer server.Close()

	client := &Client{
		HTTP:     &http.Client{Timeout: time.Second, Transport: rewriteHostTransport{base: server.URL}},
		Manifest: "cannon.json",
	}
	info, err := client.fetchGitHubLatest(
		"https://github.com/rob121/cannon/releases/download",
		"cannon",
		"https://api.github.com/repos/rob121/cannon/releases/latest",
	)
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
