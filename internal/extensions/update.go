package extensions

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/Masterminds/semver/v3"
	"github.com/rob121/cannon/internal/models"
	"github.com/rob121/cannon/internal/sites"
)

const (
	updateCheckInterval = 6 * time.Hour
	updateManifestName  = "cannon-extension.json"
)

type UpdateInfo struct {
	Version string
	URL     string
	SHA256  string
}

type updateManifest struct {
	Name          string                         `json:"name"`
	Version       string                         `json:"version"`
	LatestVersion string                         `json:"latest_version"`
	TagName       string                         `json:"tag_name"`
	AssetURL      string                         `json:"asset_url"`
	SHA256        string                         `json:"sha256"`
	Assets        map[string]updateManifestAsset `json:"assets"`
}

type updateManifestAsset struct {
	URL    string `json:"url"`
	SHA256 string `json:"sha256"`
}

type githubLatestRelease struct {
	TagName string `json:"tag_name"`
	Assets  []struct {
		Name               string `json:"name"`
		BrowserDownloadURL string `json:"browser_download_url"`
		Digest             string `json:"digest"`
	} `json:"assets"`
}

func (m *Manager) startUpdateChecker() {
	m.updateOnce.Do(func() {
		ctx := sites.WithContext(context.Background(), m.site)
		go func() {
			m.CheckUpdates(ctx)
			ticker := time.NewTicker(updateCheckInterval)
			defer ticker.Stop()
			for range ticker.C {
				m.CheckUpdates(ctx)
			}
		}()
	})
}

func (m *Manager) CheckUpdates(ctx context.Context) {
	db, err := sites.DB(ctx)
	if err != nil {
		return
	}
	var rows []models.Extension
	if err := db.Where("update_url_base <> ''").Find(&rows).Error; err != nil {
		return
	}
	for _, row := range rows {
		_ = m.CheckExtensionUpdate(ctx, row)
	}
}

func (m *Manager) CheckExtensionUpdate(ctx context.Context, row models.Extension) error {
	db, err := sites.DB(ctx)
	if err != nil {
		return err
	}
	now := time.Now()
	info, err := m.latestUpdateInfo(row)
	if err != nil {
		if dbErr := db.Model(&models.Extension{}).Where("extension_id = ?", row.ExtensionID).Updates(map[string]any{
			"update_checked_at": &now,
			"update_error":      err.Error(),
		}).Error; dbErr != nil {
			return dbErr
		}
		return err
	}
	available := newerVersion(info.Version, row.Version)
	updates := map[string]any{
		"latest_version":      info.Version,
		"update_available":    available,
		"update_asset_url":    "",
		"update_asset_sha256": "",
		"update_checked_at":   &now,
		"update_error":        "",
	}
	if available {
		updates["update_asset_url"] = info.URL
		updates["update_asset_sha256"] = info.SHA256
	}
	return db.Model(&models.Extension{}).Where("extension_id = ?", row.ExtensionID).Updates(updates).Error
}

func (m *Manager) Update(ctx context.Context, row models.Extension) error {
	db, err := sites.DB(ctx)
	if err != nil {
		return err
	}
	if err := m.CheckExtensionUpdate(ctx, row); err != nil {
		return err
	}
	if err := db.First(&row, row.ExtensionID).Error; err != nil {
		return err
	}
	if !row.UpdateAvailable {
		return fmt.Errorf("extension is already up to date")
	}
	assetURL := strings.TrimSpace(row.UpdateAssetURL)
	if assetURL == "" {
		assetURL = defaultAssetURL(row.UpdateURLBase, row.LatestVersion, row.Name)
	}
	if assetURL == "" {
		return fmt.Errorf("update asset URL is unavailable")
	}
	tmp, err := m.downloadUpdateAsset(row, assetURL)
	if err != nil {
		return err
	}
	defer os.Remove(tmp)
	if checksum := strings.TrimSpace(row.UpdateAssetSHA256); checksum != "" {
		if err := verifySHA256(tmp, checksum); err != nil {
			return err
		}
	}
	wasRunning := m.IsRunning(row.Name)
	if wasRunning {
		m.Stop(row.Name)
	}
	target := m.binaryPath(row.Name)
	if err := os.Rename(tmp, target); err != nil {
		return err
	}
	updates := map[string]any{
		"version":             row.LatestVersion,
		"update_available":    false,
		"update_asset_url":    "",
		"update_asset_sha256": "",
		"update_error":        "",
	}
	if err := db.Model(&models.Extension{}).Where("extension_id = ?", row.ExtensionID).Updates(updates).Error; err != nil {
		return err
	}
	if row.Status == models.StatusActive || wasRunning {
		if err := db.First(&row, row.ExtensionID).Error; err == nil {
			return m.Start(ctx, row)
		}
	}
	return nil
}

func (m *Manager) latestUpdateInfo(row models.Extension) (UpdateInfo, error) {
	base := strings.TrimRight(strings.TrimSpace(row.UpdateURLBase), "/")
	if base == "" {
		return UpdateInfo{}, fmt.Errorf("update URL base is empty")
	}
	if info, err := m.fetchUpdateManifest(row, base); err == nil {
		return info, nil
	}
	if apiURL := githubLatestAPIURL(base); apiURL != "" {
		if info, err := m.fetchGitHubLatest(row, apiURL); err == nil {
			return info, nil
		}
	}
	return UpdateInfo{}, fmt.Errorf("no update manifest or GitHub latest release found")
}

func (m *Manager) fetchUpdateManifest(row models.Extension, base string) (UpdateInfo, error) {
	manifestURL := base + "/latest/download/" + updateManifestName
	resp, err := m.updateClient.Get(manifestURL)
	if err != nil {
		return UpdateInfo{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return UpdateInfo{}, fmt.Errorf("manifest status %d", resp.StatusCode)
	}
	var manifest updateManifest
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&manifest); err != nil {
		return UpdateInfo{}, err
	}
	version := firstNonEmptyString(manifest.Version, manifest.LatestVersion, manifest.TagName)
	if version == "" {
		return UpdateInfo{}, fmt.Errorf("manifest version is empty")
	}
	asset := selectManifestAsset(row, manifest)
	if asset.URL == "" {
		asset.URL = defaultAssetURL(base, version, row.Name)
	}
	return UpdateInfo{Version: version, URL: asset.URL, SHA256: normalizeDigest(asset.SHA256)}, nil
}

func (m *Manager) fetchGitHubLatest(row models.Extension, apiURL string) (UpdateInfo, error) {
	resp, err := m.updateClient.Get(apiURL)
	if err != nil {
		return UpdateInfo{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return UpdateInfo{}, fmt.Errorf("github release status %d", resp.StatusCode)
	}
	var release githubLatestRelease
	if err := json.NewDecoder(io.LimitReader(resp.Body, 2<<20)).Decode(&release); err != nil {
		return UpdateInfo{}, err
	}
	version := strings.TrimSpace(release.TagName)
	if version == "" {
		return UpdateInfo{}, fmt.Errorf("github release tag is empty")
	}
	var selected struct {
		URL    string
		SHA256 string
	}
	for _, asset := range release.Assets {
		if updateAssetNameMatches(row.Name, asset.Name) {
			selected.URL = asset.BrowserDownloadURL
			selected.SHA256 = normalizeDigest(asset.Digest)
			break
		}
	}
	if selected.URL == "" {
		selected.URL = defaultAssetURL(row.UpdateURLBase, version, row.Name)
	}
	return UpdateInfo{Version: version, URL: selected.URL, SHA256: selected.SHA256}, nil
}

func selectManifestAsset(row models.Extension, manifest updateManifest) updateManifestAsset {
	for _, key := range platformKeys(row.Name) {
		if asset, ok := manifest.Assets[key]; ok && strings.TrimSpace(asset.URL) != "" {
			return asset
		}
	}
	if strings.TrimSpace(manifest.AssetURL) != "" {
		return updateManifestAsset{URL: manifest.AssetURL, SHA256: manifest.SHA256}
	}
	return updateManifestAsset{}
}

func platformKeys(name string) []string {
	return []string{
		runtime.GOOS + "_" + runtime.GOARCH,
		runtime.GOOS + "-" + runtime.GOARCH,
		name + "_" + runtime.GOOS + "_" + runtime.GOARCH,
		name + "-" + runtime.GOOS + "-" + runtime.GOARCH,
		name,
	}
}

func updateAssetNameMatches(binaryName, assetName string) bool {
	assetName = strings.TrimSpace(assetName)
	if assetName == binaryName {
		return true
	}
	for _, candidate := range platformKeys(binaryName) {
		if assetName == candidate {
			return true
		}
	}
	lower := strings.ToLower(assetName)
	return strings.Contains(lower, strings.ToLower(binaryName)) &&
		strings.Contains(lower, runtime.GOOS) &&
		strings.Contains(lower, runtime.GOARCH)
}

func defaultAssetURL(base, version, name string) string {
	base = strings.TrimRight(strings.TrimSpace(base), "/")
	version = strings.TrimSpace(version)
	name = strings.TrimSpace(name)
	if base == "" || version == "" || name == "" {
		return ""
	}
	return base + "/" + version + "/" + name
}

func githubLatestAPIURL(rawBase string) string {
	u, err := url.Parse(strings.TrimSpace(rawBase))
	if err != nil || !strings.EqualFold(u.Host, "github.com") {
		return ""
	}
	parts := strings.Split(strings.Trim(u.Path, "/"), "/")
	if len(parts) < 4 || parts[2] != "releases" || parts[3] != "download" {
		return ""
	}
	return "https://api.github.com/repos/" + path.Join(parts[0], parts[1], "releases", "latest")
}

func newerVersion(latest, current string) bool {
	latest = strings.TrimSpace(latest)
	current = strings.TrimSpace(current)
	if latest == "" {
		return false
	}
	lv, lerr := semver.NewVersion(strings.TrimPrefix(latest, "v"))
	cv, cerr := semver.NewVersion(strings.TrimPrefix(current, "v"))
	if lerr == nil && cerr == nil {
		return lv.GreaterThan(cv)
	}
	return strings.TrimPrefix(latest, "v") != strings.TrimPrefix(current, "v")
}

func normalizeDigest(value string) string {
	value = strings.TrimSpace(value)
	value = strings.TrimPrefix(value, "sha256:")
	return strings.ToLower(value)
}

func (m *Manager) downloadUpdateAsset(row models.Extension, assetURL string) (string, error) {
	resp, err := m.updateClient.Get(assetURL)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return "", fmt.Errorf("download update: status %d", resp.StatusCode)
	}
	target := m.binaryPath(row.Name)
	mode := os.FileMode(0755)
	if info, err := os.Stat(target); err == nil {
		mode = info.Mode().Perm()
	}
	tmp, err := os.CreateTemp(filepath.Dir(target), "."+row.Name+"-update-*")
	if err != nil {
		return "", err
	}
	tmpPath := tmp.Name()
	if _, err := io.Copy(tmp, resp.Body); err != nil {
		tmp.Close()
		os.Remove(tmpPath)
		return "", err
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpPath)
		return "", err
	}
	if err := os.Chmod(tmpPath, mode|0111); err != nil {
		os.Remove(tmpPath)
		return "", err
	}
	return tmpPath, nil
}

func verifySHA256(path, expected string) error {
	expected = normalizeDigest(expected)
	if expected == "" {
		return nil
	}
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return err
	}
	got := hex.EncodeToString(h.Sum(nil))
	if !strings.EqualFold(got, expected) {
		return fmt.Errorf("sha256 mismatch: got %s, want %s", got, expected)
	}
	return nil
}

func firstNonEmptyString(values ...string) string {
	for _, value := range values {
		if v := strings.TrimSpace(value); v != "" {
			return v
		}
	}
	return ""
}
