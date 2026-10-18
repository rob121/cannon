package updater

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

type githubLatestRelease struct {
	TagName string `json:"tag_name"`
	Assets  []struct {
		Name               string `json:"name"`
		BrowserDownloadURL string `json:"browser_download_url"`
		Digest             string `json:"digest"`
	} `json:"assets"`
}

// Client resolves remote release metadata.
type Client struct {
	HTTP     *http.Client
	Manifest string
}

func (c *Client) httpClient() *http.Client {
	if c.HTTP != nil {
		return c.HTTP
	}
	return http.DefaultClient
}

func (c *Client) manifestName() string {
	if strings.TrimSpace(c.Manifest) != "" {
		return c.Manifest
	}
	return "manifest.json"
}

// LatestInfo resolves the newest release for a binary from a release base URL.
func (c *Client) LatestInfo(base, binaryName string) (Info, error) {
	base = strings.TrimRight(strings.TrimSpace(base), "/")
	if base == "" {
		return Info{}, fmt.Errorf("update URL base is empty")
	}
	// Prefer the GitHub Releases API when possible. The public
	// /releases/latest/download/* CDN often keeps serving the previous
	// cannon.json for several minutes after a new release is published, and a
	// successful stale response would skip the API fallback entirely.
	if apiURL := GitHubLatestAPIURL(base); apiURL != "" {
		if info, err := c.fetchGitHubLatest(base, binaryName, apiURL); err == nil {
			return info, nil
		}
	}
	if info, err := c.fetchManifest(base, binaryName); err == nil {
		return info, nil
	}
	return Info{}, fmt.Errorf("no update manifest or GitHub latest release found")
}

func (c *Client) get(rawURL string) (*http.Response, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return nil, err
	}
	q := u.Query()
	q.Set("_", strconv.FormatInt(time.Now().UnixNano(), 10))
	u.RawQuery = q.Encode()

	req, err := http.NewRequest(http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Cache-Control", "no-cache")
	req.Header.Set("Pragma", "no-cache")
	req.Header.Set("User-Agent", "cannon-updater")
	return c.httpClient().Do(req)
}

func (c *Client) fetchManifest(base, binaryName string) (Info, error) {
	manifestURL := ManifestURL(base, c.manifestName())
	resp, err := c.get(manifestURL)
	if err != nil {
		return Info{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return Info{}, fmt.Errorf("manifest status %d", resp.StatusCode)
	}
	var manifest Manifest
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&manifest); err != nil {
		return Info{}, err
	}
	version := FirstNonEmpty(manifest.Version, manifest.LatestVersion, manifest.TagName)
	if version == "" {
		return Info{}, fmt.Errorf("manifest version is empty")
	}
	asset := SelectAsset(binaryName, manifest)
	if asset.URL == "" {
		asset.URL = DefaultAssetURL(base, version, binaryName)
	}
	return Info{Version: version, URL: asset.URL, SHA256: NormalizeDigest(asset.SHA256)}, nil
}

func (c *Client) fetchGitHubLatest(base, binaryName, apiURL string) (Info, error) {
	resp, err := c.get(apiURL)
	if err != nil {
		return Info{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return Info{}, fmt.Errorf("github release status %d", resp.StatusCode)
	}
	var release githubLatestRelease
	if err := json.NewDecoder(io.LimitReader(resp.Body, 2<<20)).Decode(&release); err != nil {
		return Info{}, err
	}
	version := strings.TrimSpace(release.TagName)
	if version == "" {
		return Info{}, fmt.Errorf("github release tag is empty")
	}
	var selected Info
	for _, asset := range release.Assets {
		if AssetNameMatches(binaryName, asset.Name) {
			selected.URL = asset.BrowserDownloadURL
			selected.SHA256 = NormalizeDigest(asset.Digest)
			break
		}
	}
	if selected.URL == "" {
		selected.URL = DefaultAssetURL(base, version, binaryName)
	}
	selected.Version = version
	return selected, nil
}
