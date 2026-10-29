package admin

import (
	"encoding/json"
	"fmt"
	_ "golang.org/x/image/webp"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"log/slog"
	"math"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/rob121/cannon/internal/config"
	"github.com/rob121/cannon/internal/hooks"
	"github.com/rob121/cannon/internal/media"
	"github.com/rob121/cannon/internal/models"
	"github.com/rob121/cannon/internal/sites"
)

const mediaBase = "/admin/media"

type mediaFolderNav struct {
	Name  string
	Label string
	Count int64
	URL   string
}

type mediaFileView struct {
	models.MediaAsset
	IsImage   bool
	CanEdit   bool
	SizeLabel string
	IconClass string
	ThumbPath string
}

func (h *Handler) media(w http.ResponseWriter, r *http.Request, path string) {
	parts := pathParts("/media", path)
	switch {
	case len(parts) == 0:
		h.mediaList(w, r)
	case parts[0] == "upload":
		h.mediaUpload(w, r)
	case parts[0] == "picker":
		h.mediaPicker(w, r)
	case parts[0] == "folders":
		h.mediaCreateFolder(w, r)
	case len(parts) == 2 && parts[1] == "delete":
		h.mediaDelete(w, r, parts[0])
	case len(parts) == 2 && parts[1] == "duplicate":
		h.mediaDuplicate(w, r, parts[0])
	case len(parts) == 2 && parts[1] == "replace":
		h.mediaReplace(w, r, parts[0])
	default:
		h.notFound(w, r)
	}
}

func (h *Handler) mediaList(w http.ResponseWriter, r *http.Request) {
	site, _ := sites.FromContext(r.Context())
	db, _ := sites.DB(r.Context())
	folder := strings.TrimSpace(r.URL.Query().Get("folder"))
	if norm, err := normalizeMediaFolder(folder); err == nil {
		folder = norm
	}
	page := parsePage(r)

	allFolders, _ := discoverMediaFolders(site, db)
	folderNav := mediaChildFolderNav(allFolders, folder, db)
	data := listPage(r, page, 0, mediaBase,
		"Browse folders and manage uploaded files.",
		"", map[string]any{"ActiveNav": "media"})
	data["FolderFilter"] = folder
	data["FolderNav"] = folderNav
	data["InFolder"] = folder != ""
	data["PageActionURL"] = mediaUploadURL(folder)
	data["PageActionLabel"] = "Upload"
	data["Uploaded"] = r.URL.Query().Get("uploaded")
	data["Duplicated"] = r.URL.Query().Get("duplicated")
	data["Edited"] = r.URL.Query().Get("edited")
	data["ParentFolderOptions"], _ = mediaFolderOptions(site, db)
	data["AllFolderNav"], _ = mediaFolderOptions(site, db)

	if folder == "" {
		var rootFiles []models.MediaAsset
		db.Where("folder = ? OR folder IS NULL", "").Order("created_at DESC").Find(&rootFiles)
		data["Rows"] = mediaFileViews(site, rootFiles)
		data["Total"] = int64(len(rootFiles))
		data["Breadcrumbs"] = mediaFolderBreadcrumbs("")
		h.render(w, r, "Media", "admin/media.html", data)
		return
	}

	q := db.Model(&models.MediaAsset{}).Where("folder = ?", folder)
	var total int64
	q.Count(&total)
	data["Total"] = total
	order := applyListSortDesc(r, data, map[string]string{"name": "name", "created": "created_at"}, "created")
	var rows []models.MediaAsset
	db.Model(&models.MediaAsset{}).Where("folder = ?", folder).
		Offset((page - 1) * pageSizeFor(r)).Limit(pageSizeFor(r)).Order(order).Find(&rows)
	data["Rows"] = mediaFileViews(site, rows)
	data["SubfolderNav"] = folderNav
	data["Breadcrumbs"] = mediaFolderBreadcrumbs(folder)
	h.render(w, r, mediaFolderLabel(folder), "admin/media.html", data)
}

func (h *Handler) mediaUpload(w http.ResponseWriter, r *http.Request) {
	site, err := sites.FromContext(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defaultFolder := strings.TrimSpace(r.URL.Query().Get("folder"))
	if defaultFolder == "" {
		defaultFolder = "content"
	}
	if norm, err := normalizeMediaFolder(defaultFolder); err == nil && norm != "" {
		defaultFolder = norm
	}
	if r.Method == http.MethodPost {
		asset, folder, err := h.saveMediaUpload(r, site, defaultFolder)
		if err != nil {
			if isMediaPickerRequest(r) {
				h.renderMediaPicker(w, r, folder, mediaPickerFilesMode(r), err.Error(), "")
			} else {
				h.renderMediaUpload(w, r, folder, err.Error())
			}
			return
		}
		if isMediaPickerRequest(r) {
			h.renderMediaPicker(w, r, folder, mediaPickerFilesMode(r), "", asset.Path)
			return
		}
		dest := mediaBase + "?uploaded=1"
		if folder != "" {
			dest = mediaBase + "?folder=" + url.QueryEscape(folder) + "&uploaded=1"
		}
		redirectList(w, r, dest)
		return
	}
	h.renderMediaUpload(w, r, defaultFolder, "")
}

func (h *Handler) saveMediaUpload(r *http.Request, site *config.SiteConfig, defaultFolder string) (models.MediaAsset, string, error) {
	mediaCfg, err := media.LoadSettings(r.Context())
	if err != nil {
		return models.MediaAsset{}, defaultFolder, err
	}
	maxMultipart := mediaCfg.MaxFileSizeBytes + (1 << 20)
	if maxMultipart < 32<<20 {
		maxMultipart = 32 << 20
	}
	if err := r.ParseMultipartForm(maxMultipart); err != nil {
		return models.MediaAsset{}, defaultFolder, err
	}
	file, header, err := r.FormFile("file")
	if err != nil {
		return models.MediaAsset{}, defaultFolder, err
	}
	defer file.Close()
	if err := media.ValidateUpload(mediaCfg, header.Filename, header.Size); err != nil {
		return models.MediaAsset{}, defaultFolder, err
	}
	folder := formString(r, "folder")
	if folder == "" {
		folder = "content"
	}
	folder, err = normalizeMediaFolder(folder)
	if err != nil {
		return models.MediaAsset{}, defaultFolder, err
	}
	if folder == "" {
		folder = "content"
	}
	destDir := filepath.Join(site.AssetsDir, "media", filepath.FromSlash(folder))
	if err := os.MkdirAll(destDir, 0o755); err != nil {
		return models.MediaAsset{}, folder, err
	}
	safeName := filepath.Base(header.Filename)
	safeName = strings.ReplaceAll(safeName, " ", "-")
	destPath := filepath.Join(destDir, safeName)
	if _, err := os.Stat(destPath); err == nil {
		ext := filepath.Ext(safeName)
		base := strings.TrimSuffix(safeName, ext)
		safeName = fmt.Sprintf("%s-%d%s", base, time.Now().Unix(), ext)
		destPath = filepath.Join(destDir, safeName)
	}
	out, err := os.Create(destPath)
	if err != nil {
		return models.MediaAsset{}, folder, err
	}
	size, err := io.Copy(out, io.LimitReader(file, mediaCfg.MaxFileSizeBytes+1))
	out.Close()
	if err != nil {
		return models.MediaAsset{}, folder, err
	}
	if mediaCfg.MaxFileSizeBytes > 0 && size > mediaCfg.MaxFileSizeBytes {
		_ = os.Remove(destPath)
		return models.MediaAsset{}, folder, fmt.Errorf("file exceeds the maximum upload size of %d MB", mediaCfg.MaxFileSizeBytes/(1024*1024))
	}
	mimeType := strings.TrimSpace(header.Header.Get("Content-Type"))
	if mimeType == "" {
		mimeType = detectMIME(destPath)
	}
	width, height := imageDimensions(destPath)
	webPath := "/assets/media/" + folder + "/" + safeName
	db, _ := sites.DB(r.Context())
	asset := models.MediaAsset{
		Name:   safeName,
		Path:   webPath,
		MIME:   mimeType,
		Size:   size,
		Folder: folder,
		Alt:    formString(r, "alt"),
		Width:  width,
		Height: height,
	}
	if err := db.Create(&asset).Error; err != nil {
		return models.MediaAsset{}, folder, err
	}
	_, _ = hooks.Fire(r.Context(), hooks.OnMediaUpload, map[string]any{
		"asset":  asset,
		"folder": folder,
	})
	if strings.HasPrefix(mimeType, "image/") && !strings.Contains(mimeType, "svg") {
		styles, _ := media.LoadStyles(r.Context())
		if err := media.GenerateStyles(destPath, styles); err != nil {
			slog.Warn("media image styles", "path", destPath, "err", err)
		}
	}
	return asset, folder, nil
}

func (h *Handler) renderMediaUpload(w http.ResponseWriter, r *http.Request, defaultFolder, errMsg string) {
	site, _ := sites.FromContext(r.Context())
	db, _ := sites.DB(r.Context())
	folderOptions, _ := mediaFolderOptions(site, db)
	mediaCfg, _ := media.LoadSettings(r.Context())
	maxMB := int64(32)
	if mediaCfg.MaxFileSizeBytes > 0 {
		maxMB = mediaCfg.MaxFileSizeBytes / (1024 * 1024)
	}
	data := formData(map[string]any{
		"ActiveNav":           "media",
		"BasePath":            mediaBase,
		"CancelURL":           mediaFolderURL(defaultFolder),
		"DefaultFolder":       defaultFolder,
		"FolderOptions":       folderOptions,
		"ParentFolderOptions": folderOptions,
		"MaxFileSizeMB":       maxMB,
		"ApprovedExtensions":  mediaCfg.ApprovedExtensionsLabel(),
		"Subtitle":            "Upload images and files to the media library.",
		"Title":               "Upload Media",
	})
	if errMsg != "" {
		data["Error"] = errMsg
	}
	h.render(w, r, "Upload Media", "admin/media_upload.html", data)
}

func (h *Handler) mediaCreateFolder(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	site, err := sites.FromContext(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	parent := formString(r, "parent")
	name := formString(r, "name")
	folder, err := joinMediaFolder(parent, name)
	if err != nil {
		h.renderMediaFolderError(w, r, parent, err.Error())
		return
	}
	if err := ensureMediaFolder(site, folder); err != nil {
		h.renderMediaFolderError(w, r, parent, err.Error())
		return
	}
	returnTo := formString(r, "return")
	switch returnTo {
	case "upload":
		redirectList(w, r, mediaUploadURL(folder))
	case "browse":
		redirectList(w, r, mediaFolderURL(folder))
	case "picker":
		h.renderMediaPicker(w, r, folder, mediaPickerFilesMode(r), "", "")
	default:
		if formString(r, "from") == "upload" {
			redirectList(w, r, mediaUploadURL(folder))
			return
		}
		redirectList(w, r, mediaFolderURL(folder))
	}
}

func (h *Handler) renderMediaFolderError(w http.ResponseWriter, r *http.Request, parent, errMsg string) {
	site, _ := sites.FromContext(r.Context())
	db, _ := sites.DB(r.Context())
	folderOptions, _ := mediaFolderOptions(site, db)
	if formString(r, "return") == "picker" {
		h.renderMediaPicker(w, r, parent, mediaPickerFilesMode(r), errMsg, "")
		return
	}
	if formString(r, "from") == "upload" || formString(r, "return") == "upload" {
		data := formData(map[string]any{
			"ActiveNav":           "media",
			"BasePath":            mediaBase,
			"CancelURL":           mediaFolderURL(parent),
			"DefaultFolder":       parent,
			"FolderOptions":       folderOptions,
			"ParentFolderOptions": folderOptions,
			"NewFolderParent":     parent,
			"NewFolderName":       formString(r, "name"),
			"Subtitle":            "Upload images and files to the media library.",
			"Title":               "Upload Media",
			"Error":               errMsg,
		})
		h.render(w, r, "Upload Media", "admin/media_upload.html", data)
		return
	}
	folder := parent
	data := listPage(r, 1, 0, mediaBase,
		"Browse folders and manage uploaded files.",
		"", map[string]any{"ActiveNav": "media"})
	data["FolderFilter"] = folder
	data["InFolder"] = folder != ""
	data["ParentFolderOptions"] = folderOptions
	data["NewFolderParent"] = parent
	data["NewFolderName"] = formString(r, "name")
	data["Breadcrumbs"] = mediaFolderBreadcrumbs(folder)
	data["Error"] = errMsg
	allFolders, _ := discoverMediaFolders(site, db)
	data["FolderNav"] = mediaChildFolderNav(allFolders, folder, db)
	data["SubfolderNav"] = data["FolderNav"]
	h.render(w, r, "Media", "admin/media.html", data)
}

func (h *Handler) mediaDelete(w http.ResponseWriter, r *http.Request, idStr string) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	id, ok := parseID(idStr)
	if !ok {
		h.notFound(w, r)
		return
	}
	site, _ := sites.FromContext(r.Context())
	db, _ := sites.DB(r.Context())
	var asset models.MediaAsset
	if err := db.First(&asset, id).Error; err != nil {
		h.notFound(w, r)
		return
	}
	if _, err := hooks.Fire(r.Context(), hooks.OnMediaDelete, map[string]any{
		"asset_id": id,
		"asset":    asset,
	}); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if site != nil {
		local := filepath.Join(site.AssetsDir, strings.TrimPrefix(asset.Path, "/assets/"))
		styles, _ := media.LoadStyles(r.Context())
		media.RemoveDerivatives(local, styles)
		_ = os.Remove(local)
	}
	if err := db.Delete(&models.MediaAsset{}, id).Error; err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	redirectList(w, r, mediaBase+listRedirectQuery(r))
}

func (h *Handler) mediaDuplicate(w http.ResponseWriter, r *http.Request, idStr string) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	id, ok := parseID(idStr)
	if !ok {
		h.notFound(w, r)
		return
	}
	site, err := sites.FromContext(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	db, _ := sites.DB(r.Context())
	var asset models.MediaAsset
	if err := db.First(&asset, id).Error; err != nil {
		h.notFound(w, r)
		return
	}
	local := filepath.Join(site.AssetsDir, strings.TrimPrefix(asset.Path, "/assets/"))
	src, err := os.Open(local)
	if err != nil {
		http.Error(w, "file not found on disk", http.StatusNotFound)
		return
	}
	defer src.Close()

	folder := strings.TrimSpace(asset.Folder)
	if folder == "" {
		folder = "content"
	}
	destDir := filepath.Join(site.AssetsDir, "media", filepath.FromSlash(folder))
	if err := os.MkdirAll(destDir, 0o755); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	newName := uniqueMediaFilename(destDir, duplicateMediaFilename(asset.Name))
	destPath := filepath.Join(destDir, newName)
	out, err := os.Create(destPath)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	size, err := io.Copy(out, src)
	out.Close()
	if err != nil {
		_ = os.Remove(destPath)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	mimeType := detectMIME(destPath)
	width, height := imageDimensions(destPath)
	webPath := "/assets/media/" + folder + "/" + newName
	copy := models.MediaAsset{
		Name:   newName,
		Path:   webPath,
		MIME:   mimeType,
		Size:   size,
		Folder: folder,
		Alt:    asset.Alt,
		Width:  width,
		Height: height,
	}
	if err := db.Create(&copy).Error; err != nil {
		_ = os.Remove(destPath)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if strings.HasPrefix(mimeType, "image/") && !strings.Contains(mimeType, "svg") {
		styles, _ := media.LoadStyles(r.Context())
		if err := media.GenerateStyles(destPath, styles); err != nil {
			slog.Warn("media duplicate styles", "path", destPath, "err", err)
		}
	}
	dest := mediaBase + "?duplicated=1"
	if folder != "" {
		dest = mediaBase + "?folder=" + url.QueryEscape(folder) + "&duplicated=1"
	}
	redirectList(w, r, dest)
}

func (h *Handler) mediaReplace(w http.ResponseWriter, r *http.Request, idStr string) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	id, ok := parseID(idStr)
	if !ok {
		h.notFound(w, r)
		return
	}
	site, err := sites.FromContext(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	mediaCfg, err := media.LoadSettings(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	maxMultipart := mediaCfg.MaxFileSizeBytes + (1 << 20)
	if maxMultipart < 32<<20 {
		maxMultipart = 32 << 20
	}
	if err := r.ParseMultipartForm(maxMultipart); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	file, header, err := r.FormFile("file")
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	defer file.Close()
	if err := media.ValidateUpload(mediaCfg, header.Filename, header.Size); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	db, _ := sites.DB(r.Context())
	var asset models.MediaAsset
	if err := db.First(&asset, id).Error; err != nil {
		h.notFound(w, r)
		return
	}
	local := filepath.Join(site.AssetsDir, strings.TrimPrefix(asset.Path, "/assets/"))
	styles, _ := media.LoadStyles(r.Context())
	media.RemoveDerivatives(local, styles)

	out, err := os.Create(local)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	size, err := io.Copy(out, io.LimitReader(file, mediaCfg.MaxFileSizeBytes+1))
	out.Close()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if mediaCfg.MaxFileSizeBytes > 0 && size > mediaCfg.MaxFileSizeBytes {
		_ = os.Remove(local)
		http.Error(w, fmt.Sprintf("file exceeds the maximum upload size of %d MB", mediaCfg.MaxFileSizeBytes/(1024*1024)), http.StatusBadRequest)
		return
	}
	mimeType := detectMIME(local)
	if !isEditableImageMIME(mimeType) {
		_ = os.Remove(local)
		http.Error(w, "only JPEG, PNG, and WebP images can be saved from the editor", http.StatusBadRequest)
		return
	}
	width, height := imageDimensions(local)
	asset.Size = size
	asset.MIME = mimeType
	asset.Width = width
	asset.Height = height
	if err := db.Save(&asset).Error; err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if wantsJSONResponse(r) {
		writeJSON(w, http.StatusOK, map[string]any{
			"ok":     true,
			"path":   asset.Path,
			"mime":   mimeType,
			"width":  width,
			"height": height,
			"size":   size,
		})
		localPath := local
		styleSet := styles
		go func() {
			if err := media.GenerateStyles(localPath, styleSet); err != nil {
				slog.Warn("media replace styles", "path", localPath, "err", err)
			}
		}()
		return
	}
	if err := media.GenerateStyles(local, styles); err != nil {
		slog.Warn("media replace styles", "path", local, "err", err)
	}
	dest := mediaBase + "?edited=1"
	if asset.Folder != "" {
		dest = mediaBase + "?folder=" + url.QueryEscape(asset.Folder) + "&edited=1"
	}
	redirectList(w, r, dest)
}

func duplicateMediaFilename(name string) string {
	ext := filepath.Ext(name)
	base := strings.TrimSuffix(filepath.Base(name), ext)
	base = strings.TrimSpace(base)
	if base == "" {
		base = "copy"
	}
	return base + "-copy" + ext
}

func uniqueMediaFilename(dir, name string) string {
	safeName := filepath.Base(strings.ReplaceAll(name, " ", "-"))
	destPath := filepath.Join(dir, safeName)
	if _, err := os.Stat(destPath); os.IsNotExist(err) {
		return safeName
	}
	ext := filepath.Ext(safeName)
	base := strings.TrimSuffix(safeName, ext)
	return fmt.Sprintf("%s-%d%s", base, time.Now().Unix(), ext)
}

func isEditableImageMIME(mime string) bool {
	switch strings.ToLower(strings.TrimSpace(mime)) {
	case "image/jpeg", "image/png", "image/webp":
		return true
	default:
		return false
	}
}

func mediaFileViews(site *config.SiteConfig, rows []models.MediaAsset) []mediaFileView {
	out := make([]mediaFileView, 0, len(rows))
	assetsDir := ""
	if site != nil {
		assetsDir = site.AssetsDir
	}
	for _, row := range rows {
		view := mediaFileView{
			MediaAsset: row,
			IsImage:    isImageMIME(row.MIME),
			CanEdit:    isEditableImageMIME(row.MIME),
			SizeLabel:  formatFileSize(row.Size),
			IconClass:  mediaIconClass(row.MIME),
		}
		if view.IsImage {
			view.ThumbPath = media.ResolveWebThumbnail(assetsDir, row.Path)
		}
		out = append(out, view)
	}
	return out
}

func mediaFolderURL(folder string) string {
	if strings.TrimSpace(folder) == "" {
		return mediaBase
	}
	return mediaBase + "?folder=" + url.QueryEscape(folder)
}

func mediaUploadURL(folder string) string {
	if strings.TrimSpace(folder) == "" {
		return mediaBase + "/upload"
	}
	return mediaBase + "/upload?folder=" + url.QueryEscape(folder)
}

func mediaFolderLabel(folder string) string {
	folder = strings.TrimSpace(folder)
	if folder == "" {
		return "All Files"
	}
	return strings.ReplaceAll(folder, "-", " ")
}

func formatFileSize(size int64) string {
	if size < 1024 {
		return fmt.Sprintf("%d B", size)
	}
	units := []string{"KB", "MB", "GB", "TB"}
	value := float64(size)
	for _, unit := range units {
		value /= 1024
		if value < 1024 {
			if value >= 100 || math.Abs(value-math.Round(value)) < 0.05 {
				return fmt.Sprintf("%.0f %s", value, unit)
			}
			return fmt.Sprintf("%.1f %s", value, unit)
		}
	}
	return fmt.Sprintf("%.1f PB", value/1024)
}

func isImageMIME(mime string) bool {
	return strings.HasPrefix(strings.ToLower(strings.TrimSpace(mime)), "image/")
}

func mediaIconClass(mime string) string {
	mime = strings.ToLower(strings.TrimSpace(mime))
	switch {
	case strings.HasPrefix(mime, "image/"):
		return "bi-file-earmark-image"
	case strings.HasPrefix(mime, "video/"):
		return "bi-file-earmark-play"
	case strings.HasPrefix(mime, "audio/"):
		return "bi-file-earmark-music"
	case mime == "application/pdf":
		return "bi-file-earmark-pdf"
	case strings.HasPrefix(mime, "text/"):
		return "bi-file-earmark-text"
	default:
		return "bi-file-earmark"
	}
}

func detectMIME(path string) string {
	f, err := os.Open(path)
	if err != nil {
		return "application/octet-stream"
	}
	defer f.Close()
	buf := make([]byte, 512)
	n, _ := f.Read(buf)
	if n == 0 {
		return "application/octet-stream"
	}
	return http.DetectContentType(buf[:n])
}

func imageDimensions(path string) (int, int) {
	f, err := os.Open(path)
	if err != nil {
		return 0, 0
	}
	defer f.Close()
	cfg, _, err := image.DecodeConfig(f)
	if err != nil {
		return 0, 0
	}
	return cfg.Width, cfg.Height
}

func (h *Handler) mediaPicker(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	folder := strings.TrimSpace(r.URL.Query().Get("folder"))
	if norm, err := normalizeMediaFolder(folder); err == nil {
		folder = norm
	}
	h.renderMediaPicker(w, r, folder, mediaPickerFilesMode(r), "", "")
}

func (h *Handler) renderMediaPicker(w http.ResponseWriter, r *http.Request, folder string, filesMode bool, errMsg, uploadedPath string) {
	site, _ := sites.FromContext(r.Context())
	db, _ := sites.DB(r.Context())
	if norm, err := normalizeMediaFolder(folder); err == nil {
		folder = norm
	}
	q := db.Model(&models.MediaAsset{})
	if !filesMode {
		q = q.Where("mime LIKE ?", "image/%")
	}
	if folder != "" {
		q = q.Where("folder = ?", folder)
	}
	var rows []models.MediaAsset
	q.Order("created_at DESC").Limit(120).Find(&rows)
	folderOptions, _ := mediaFolderOptions(site, db)
	mediaCfg, _ := media.LoadSettings(r.Context())
	maxMB := int64(32)
	if mediaCfg.MaxFileSizeBytes > 0 {
		maxMB = mediaCfg.MaxFileSizeBytes / (1024 * 1024)
	}
	uploadFolder := folder
	if uploadFolder == "" {
		uploadFolder = "content"
	}
	h.renderFragment(w, r, "admin/media_picker.html", map[string]any{
		"BasePath":            mediaBase,
		"FolderFilter":        folder,
		"FolderOptions":       folderOptions,
		"ParentFolderOptions": folderOptions,
		"DefaultFolder":       uploadFolder,
		"Rows":                mediaFileViews(site, rows),
		"FilesMode":           filesMode,
		"PickerType":          mediaPickerType(filesMode),
		"MaxFileSizeMB":       maxMB,
		"ApprovedExtensions":  mediaCfg.ApprovedExtensionsLabel(),
		"Error":               errMsg,
		"UploadedPath":        uploadedPath,
	})
}

func isMediaPickerRequest(r *http.Request) bool {
	return strings.TrimSpace(r.URL.Query().Get("picker")) == "1" || strings.TrimSpace(r.FormValue("picker")) == "1"
}

func mediaPickerFilesMode(r *http.Request) bool {
	return strings.TrimSpace(r.URL.Query().Get("type")) == "files" || strings.TrimSpace(r.FormValue("type")) == "files"
}

func mediaPickerType(filesMode bool) string {
	if filesMode {
		return "files"
	}
	return "images"
}

func wantsJSONResponse(r *http.Request) bool {
	if strings.EqualFold(r.Header.Get("X-Requested-With"), "XMLHttpRequest") {
		return true
	}
	return strings.Contains(r.Header.Get("Accept"), "application/json")
}

func writeJSON(w http.ResponseWriter, code int, payload any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(payload)
}
