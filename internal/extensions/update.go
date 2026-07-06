package extensions

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/rob121/cannon/internal/models"
	"github.com/rob121/cannon/internal/sites"
	"github.com/rob121/cannon/internal/updater"
)

const (
	updateCheckInterval = 6 * time.Hour
	updateManifestName  = "cannon-extension.json"
)

type UpdateInfo = updater.Info

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
	available := updater.NewerVersion(info.Version, row.Version)
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
		assetURL = updater.DefaultAssetURL(row.UpdateURLBase, row.LatestVersion, row.Name)
	}
	if assetURL == "" {
		return fmt.Errorf("update asset URL is unavailable")
	}
	target := m.binaryPath(row.Name)
	tmp, err := updater.Download(m.updateClient, assetURL, target)
	if err != nil {
		return err
	}
	defer os.Remove(tmp)
	if checksum := strings.TrimSpace(row.UpdateAssetSHA256); checksum != "" {
		if err := updater.VerifySHA256(tmp, checksum); err != nil {
			return err
		}
	}
	wasRunning := m.IsRunning(row.Name)
	if wasRunning {
		m.Stop(row.Name)
	}
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
	client := &updater.Client{HTTP: m.updateClient, Manifest: updateManifestName}
	return client.LatestInfo(row.UpdateURLBase, row.Name)
}
