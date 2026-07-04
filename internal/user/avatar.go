package user

import (
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/rob121/cannon/internal/config"
	"github.com/rob121/cannon/internal/media"
	"github.com/rob121/cannon/internal/models"
	"github.com/rob121/cannon/internal/sites"
)

const avatarMediaFolder = "avatars"

// ResolveAvatar returns the best avatar URL for a user: uploaded, then SSO.
func ResolveAvatar(u *models.User) string {
	if u == nil {
		return ""
	}
	if v := strings.TrimSpace(u.AvatarURL); v != "" {
		return v
	}
	return strings.TrimSpace(u.SSOAvatarURL)
}

// SaveAvatarUpload stores an uploaded avatar image and returns its public path.
func SaveAvatarUpload(ctx context.Context, site *config.SiteConfig, userID uint, file multipart.File, header *multipart.FileHeader) (string, error) {
	if site == nil || userID == 0 || file == nil || header == nil {
		return "", fmt.Errorf("avatar upload is invalid")
	}
	cfg, err := media.LoadSettings(ctx)
	if err != nil {
		return "", err
	}
	if err := media.ValidateUpload(cfg, header.Filename, header.Size); err != nil {
		return "", err
	}
	ext := strings.ToLower(strings.TrimPrefix(filepath.Ext(header.Filename), "."))
	if ext == "" {
		return "", fmt.Errorf("avatar must be an image file")
	}
	allowedImage := false
	for _, allowed := range []string{"jpg", "jpeg", "png", "gif", "webp", "svg"} {
		if ext == allowed {
			allowedImage = true
			break
		}
	}
	if !allowedImage {
		return "", fmt.Errorf("avatar must be a JPG, PNG, GIF, WebP, or SVG image")
	}

	destDir := filepath.Join(site.AssetsDir, "media", avatarMediaFolder)
	if err := os.MkdirAll(destDir, 0o755); err != nil {
		return "", err
	}
	safeName := fmt.Sprintf("user-%d-%d.%s", userID, time.Now().Unix(), ext)
	destPath := filepath.Join(destDir, safeName)
	out, err := os.Create(destPath)
	if err != nil {
		return "", err
	}
	size, err := io.Copy(out, io.LimitReader(file, cfg.MaxFileSizeBytes+1))
	out.Close()
	if err != nil {
		_ = os.Remove(destPath)
		return "", err
	}
	if cfg.MaxFileSizeBytes > 0 && size > cfg.MaxFileSizeBytes {
		_ = os.Remove(destPath)
		return "", fmt.Errorf("avatar exceeds the maximum upload size")
	}
	webPath := "/assets/media/" + avatarMediaFolder + "/" + safeName
	if strings.HasPrefix(header.Header.Get("Content-Type"), "image/") {
		_, _ = media.GenerateThumbnail(destPath)
	}
	return webPath, nil
}

// SaveProfileFieldImage stores an uploaded profile field image and returns its public path.
func SaveProfileFieldImage(ctx context.Context, site *config.SiteConfig, userID, fieldID uint, file multipart.File, header *multipart.FileHeader) (string, error) {
	if site == nil || userID == 0 || fieldID == 0 || file == nil || header == nil {
		return "", fmt.Errorf("profile image upload is invalid")
	}
	cfg, err := media.LoadSettings(ctx)
	if err != nil {
		return "", err
	}
	if err := media.ValidateUpload(cfg, header.Filename, header.Size); err != nil {
		return "", err
	}
	ext := strings.ToLower(strings.TrimPrefix(filepath.Ext(header.Filename), "."))
	if ext == "" {
		return "", fmt.Errorf("image must be an image file")
	}
	allowedImage := false
	for _, allowed := range []string{"jpg", "jpeg", "png", "gif", "webp", "svg"} {
		if ext == allowed {
			allowedImage = true
			break
		}
	}
	if !allowedImage {
		return "", fmt.Errorf("image must be a JPG, PNG, GIF, WebP, or SVG file")
	}

	folder := filepath.Join("profile", fmt.Sprintf("user-%d", userID))
	destDir := filepath.Join(site.AssetsDir, "media", folder)
	if err := os.MkdirAll(destDir, 0o755); err != nil {
		return "", err
	}
	safeName := fmt.Sprintf("field-%d-%d.%s", fieldID, time.Now().Unix(), ext)
	destPath := filepath.Join(destDir, safeName)
	out, err := os.Create(destPath)
	if err != nil {
		return "", err
	}
	size, err := io.Copy(out, io.LimitReader(file, cfg.MaxFileSizeBytes+1))
	out.Close()
	if err != nil {
		_ = os.Remove(destPath)
		return "", err
	}
	if cfg.MaxFileSizeBytes > 0 && size > cfg.MaxFileSizeBytes {
		_ = os.Remove(destPath)
		return "", fmt.Errorf("image exceeds the maximum upload size")
	}
	webPath := "/assets/media/" + folder + "/" + safeName
	if strings.HasPrefix(header.Header.Get("Content-Type"), "image/") {
		_, _ = media.GenerateThumbnail(destPath)
	}
	return webPath, nil
}

// UpdateAvatarURL sets the user's uploaded avatar path.
func UpdateAvatarURL(ctx context.Context, userID uint, avatarURL string) error {
	db, err := sites.DB(ctx)
	if err != nil {
		return err
	}
	avatarURL = strings.TrimSpace(avatarURL)
	return db.Model(&models.User{}).Where("user_id = ?", userID).Update("avatar_url", avatarURL).Error
}

// ClearAvatar removes the uploaded avatar and deletes the local file when possible.
func ClearAvatar(ctx context.Context, site *config.SiteConfig, userID uint) error {
	db, err := sites.DB(ctx)
	if err != nil {
		return err
	}
	var u models.User
	if err := db.First(&u, userID).Error; err != nil {
		return err
	}
	if site != nil {
		removeLocalAsset(site.AssetsDir, u.AvatarURL)
	}
	return db.Model(&u).Update("avatar_url", "").Error
}

// SyncSSOAvatar stores the provider avatar when the user has not uploaded one.
func SyncSSOAvatar(ctx context.Context, userID uint, avatarURL string) error {
	avatarURL = strings.TrimSpace(avatarURL)
	if userID == 0 || avatarURL == "" {
		return nil
	}
	db, err := sites.DB(ctx)
	if err != nil {
		return err
	}
	return db.Model(&models.User{}).Where("user_id = ?", userID).Update("sso_avatar_url", avatarURL).Error
}

func removeLocalAsset(assetsDir, webPath string) {
	webPath = strings.TrimSpace(webPath)
	if assetsDir == "" || webPath == "" || !strings.HasPrefix(webPath, "/assets/") {
		return
	}
	local := filepath.Join(assetsDir, strings.TrimPrefix(webPath, "/assets/"))
	_ = os.Remove(local)
	thumb := strings.TrimSuffix(local, filepath.Ext(local)) + "_thumb" + filepath.Ext(local)
	_ = os.Remove(thumb)
}
