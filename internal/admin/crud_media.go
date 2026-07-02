package admin

import (
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"math"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

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
	data["ParentFolderOptions"], _ = mediaFolderOptions(site, db)
	data["AllFolderNav"], _ = mediaFolderOptions(site, db)

	if folder == "" {
		var rootFiles []models.MediaAsset
		db.Where("folder = ? OR folder IS NULL", "").Order("created_at DESC").Find(&rootFiles)
		data["Rows"] = mediaFileViews(rootFiles)
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
	data["Rows"] = mediaFileViews(rows)
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
		mediaCfg, err := media.LoadSettings(r.Context())
		if err != nil {
			h.renderMediaUpload(w, r, defaultFolder, err.Error())
			return
		}
		maxMultipart := mediaCfg.MaxFileSizeBytes + (1 << 20)
		if maxMultipart < 32<<20 {
			maxMultipart = 32 << 20
		}
		if err := r.ParseMultipartForm(maxMultipart); err != nil {
			h.renderMediaUpload(w, r, defaultFolder, err.Error())
			return
		}
		file, header, err := r.FormFile("file")
		if err != nil {
			h.renderMediaUpload(w, r, defaultFolder, err.Error())
			return
		}
		defer file.Close()
		if err := media.ValidateUpload(mediaCfg, header.Filename, header.Size); err != nil {
			h.renderMediaUpload(w, r, defaultFolder, err.Error())
			return
		}
		folder := formString(r, "folder")
		if folder == "" {
			folder = "content"
		}
		folder, err = normalizeMediaFolder(folder)
		if err != nil {
			h.renderMediaUpload(w, r, defaultFolder, err.Error())
			return
		}
		if folder == "" {
			folder = "content"
		}
		destDir := filepath.Join(site.AssetsDir, "media", filepath.FromSlash(folder))
		if err := os.MkdirAll(destDir, 0o755); err != nil {
			h.renderMediaUpload(w, r, folder, err.Error())
			return
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
			h.renderMediaUpload(w, r, folder, err.Error())
			return
		}
		size, err := io.Copy(out, io.LimitReader(file, mediaCfg.MaxFileSizeBytes+1))
		out.Close()
		if err != nil {
			h.renderMediaUpload(w, r, folder, err.Error())
			return
		}
		if mediaCfg.MaxFileSizeBytes > 0 && size > mediaCfg.MaxFileSizeBytes {
			_ = os.Remove(destPath)
			h.renderMediaUpload(w, r, folder, fmt.Sprintf("file exceeds the maximum upload size of %d MB", mediaCfg.MaxFileSizeBytes/(1024*1024)))
			return
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
			h.renderMediaUpload(w, r, folder, err.Error())
			return
		}
		if strings.HasPrefix(mimeType, "image/") {
			_, _ = media.GenerateThumbnail(destPath)
		}
		dest := mediaBase
		if folder != "" {
			dest = mediaFolderURL(folder)
		}
		redirectList(w, r, dest)
		return
	}
	h.renderMediaUpload(w, r, defaultFolder, "")
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
		"ActiveNav":            "media",
		"BasePath":             mediaBase,
		"CancelURL":            mediaFolderURL(defaultFolder),
		"DefaultFolder":        defaultFolder,
		"FolderOptions":        folderOptions,
		"ParentFolderOptions":  folderOptions,
		"MaxFileSizeMB":        maxMB,
		"ApprovedExtensions":   mediaCfg.ApprovedExtensionsLabel(),
		"Subtitle":             "Upload images and files to the media library.",
		"Title":                "Upload Media",
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
	if err := db.First(&asset, id).Error; err == nil && site != nil {
		local := strings.TrimPrefix(asset.Path, "/assets/")
		_ = os.Remove(filepath.Join(site.AssetsDir, local))
	}
	if err := db.Delete(&models.MediaAsset{}, id).Error; err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	redirectList(w, r, mediaBase+listRedirectQuery(r))
}

func mediaFileViews(rows []models.MediaAsset) []mediaFileView {
	out := make([]mediaFileView, 0, len(rows))
	for _, row := range rows {
		view := mediaFileView{
			MediaAsset: row,
			IsImage:    isImageMIME(row.MIME),
			SizeLabel:  formatFileSize(row.Size),
			IconClass:  mediaIconClass(row.MIME),
		}
		if view.IsImage {
			view.ThumbPath = media.WebThumbnailPath(row.Path)
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
	site, _ := sites.FromContext(r.Context())
	db, _ := sites.DB(r.Context())
	folder := strings.TrimSpace(r.URL.Query().Get("folder"))
	if norm, err := normalizeMediaFolder(folder); err == nil {
		folder = norm
	}
	q := db.Model(&models.MediaAsset{}).Where("mime LIKE ?", "image/%")
	if folder != "" {
		q = q.Where("folder = ?", folder)
	}
	var rows []models.MediaAsset
	q.Order("created_at DESC").Limit(120).Find(&rows)
	folderOptions, _ := mediaFolderOptions(site, db)
	h.renderFragment(w, r, "admin/media_picker.html", map[string]any{
		"BasePath":      mediaBase,
		"FolderFilter":  folder,
		"FolderOptions": folderOptions,
		"Rows":          mediaFileViews(rows),
	})
}
