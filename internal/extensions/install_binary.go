package extensions

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/rob121/cannon/internal/models"
	"github.com/rob121/cannon/internal/sites"
)

const MaxExtensionBinaryBytes int64 = 200 << 20

var extensionBinaryNamePattern = regexp.MustCompile(`^[A-Za-z0-9._-]+$`)

type SaveBinaryOptions struct {
	Name          string
	Source        io.Reader
	SHA256        string
	UpdateURLBase string
	LatestVersion string
}

func (m *Manager) SaveBinary(ctx context.Context, opts SaveBinaryOptions) (models.Extension, error) {
	name, err := normalizeExtensionBinaryName(opts.Name)
	if err != nil {
		return models.Extension{}, err
	}
	if opts.Source == nil {
		return models.Extension{}, fmt.Errorf("extension binary source is required")
	}
	if err := os.MkdirAll(m.app.Extensions.Dir, 0755); err != nil {
		return models.Extension{}, err
	}
	if err := os.MkdirAll(m.app.Extensions.SocketsDir, 0755); err != nil {
		return models.Extension{}, err
	}

	tmp, err := os.CreateTemp(m.app.Extensions.Dir, "."+name+"-upload-*")
	if err != nil {
		return models.Extension{}, err
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)

	h := sha256.New()
	limited := io.LimitReader(opts.Source, MaxExtensionBinaryBytes+1)
	n, copyErr := io.Copy(io.MultiWriter(tmp, h), limited)
	closeErr := tmp.Close()
	if copyErr != nil {
		return models.Extension{}, copyErr
	}
	if closeErr != nil {
		return models.Extension{}, closeErr
	}
	if n > MaxExtensionBinaryBytes {
		return models.Extension{}, fmt.Errorf("extension binary exceeds the %d MB limit", MaxExtensionBinaryBytes/(1024*1024))
	}
	if expected := normalizeDigest(opts.SHA256); expected != "" {
		got := hex.EncodeToString(h.Sum(nil))
		if !strings.EqualFold(got, expected) {
			return models.Extension{}, fmt.Errorf("sha256 mismatch: got %s, want %s", got, expected)
		}
	}
	if err := os.Chmod(tmpPath, 0755); err != nil {
		return models.Extension{}, err
	}
	if err := os.Rename(tmpPath, m.binaryPath(name)); err != nil {
		return models.Extension{}, err
	}

	db, err := sites.DB(ctx)
	if err != nil {
		return models.Extension{}, err
	}
	if err := m.syncDirectory(db); err != nil {
		return models.Extension{}, err
	}
	var row models.Extension
	if err := db.Where("name = ?", name).First(&row).Error; err != nil {
		return models.Extension{}, err
	}
	updates := map[string]any{"update_error": ""}
	if v := strings.TrimSpace(opts.UpdateURLBase); v != "" {
		updates["update_url_base"] = v
		row.UpdateURLBase = v
	}
	if v := strings.TrimSpace(opts.LatestVersion); v != "" {
		updates["latest_version"] = v
		updates["update_available"] = false
		updates["update_asset_url"] = ""
		updates["update_asset_sha256"] = ""
		row.LatestVersion = v
		row.UpdateAvailable = false
		row.UpdateAssetURL = ""
		row.UpdateAssetSHA256 = ""
	}
	if err := db.Model(&models.Extension{}).Where("extension_id = ?", row.ExtensionID).Updates(updates).Error; err != nil {
		return models.Extension{}, err
	}
	if err := db.First(&row, row.ExtensionID).Error; err != nil {
		return models.Extension{}, err
	}
	return row, nil
}

// NormalizeBinaryName validates and normalizes an extension binary filename.
func NormalizeBinaryName(name string) (string, error) {
	return normalizeExtensionBinaryName(name)
}

func normalizeExtensionBinaryName(name string) (string, error) {
	name = strings.TrimSpace(name)
	name = filepath.Base(name)
	if name == "." || name == string(filepath.Separator) || name == "" || strings.HasPrefix(name, ".") {
		return "", fmt.Errorf("extension binary name is required")
	}
	if !extensionBinaryNamePattern.MatchString(name) {
		return "", fmt.Errorf("extension binary name %q may only contain letters, numbers, dots, underscores, and hyphens", name)
	}
	return name, nil
}
