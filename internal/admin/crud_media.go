package admin

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/rob121/cannon/internal/models"
	"github.com/rob121/cannon/internal/sites"
)

const mediaBase = "/admin/media"

func (h *Handler) media(w http.ResponseWriter, r *http.Request, path string) {
	parts := pathParts("/media", path)
	switch {
	case len(parts) == 0:
		h.mediaList(w, r)
	case parts[0] == "upload":
		h.mediaUpload(w, r)
	case len(parts) == 2 && parts[1] == "delete":
		h.mediaDelete(w, r, parts[0])
	default:
		http.NotFound(w, r)
	}
}

func (h *Handler) mediaList(w http.ResponseWriter, r *http.Request) {
	db, _ := sites.DB(r.Context())
	page := parsePage(r)
	folder := r.URL.Query().Get("folder")
	q := db.Model(&models.MediaAsset{})
	if folder != "" {
		q = q.Where("folder = ?", folder)
	}
	var total int64
	q.Count(&total)
	data := listPage(page, total, mediaBase,
		"Upload and manage images and files.",
		"", map[string]any{"ActiveNav": "media"})
	order := applyListSort(r, data, map[string]string{"name": "name", "created": "created_at"}, "created")
	var rows []models.MediaAsset
	q.Offset((page - 1) * pageSize).Limit(pageSize).Order(order + " DESC").Find(&rows)
	var folders []string
	db.Model(&models.MediaAsset{}).Distinct("folder").Pluck("folder", &folders)
	data["Rows"] = rows
	data["FolderFilter"] = folder
	data["Folders"] = folders
	data["PageActionURL"] = mediaBase + "/upload"
	data["PageActionLabel"] = "Upload File"
	h.render(w, r, "Media", "admin/media.html", data)
}

func (h *Handler) mediaUpload(w http.ResponseWriter, r *http.Request) {
	site, err := sites.FromContext(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if r.Method == http.MethodPost {
		if err := r.ParseMultipartForm(32 << 20); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		file, header, err := r.FormFile("file")
		if err != nil {
			h.renderMediaUpload(w, r, err.Error())
			return
		}
		defer file.Close()
		folder := formString(r, "folder")
		if folder == "" {
			folder = "content"
		}
		folder = strings.Trim(folder, "/")
		destDir := filepath.Join(site.AssetsDir, "media", folder)
		if err := os.MkdirAll(destDir, 0o755); err != nil {
			h.renderMediaUpload(w, r, err.Error())
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
			h.renderMediaUpload(w, r, err.Error())
			return
		}
		size, err := io.Copy(out, file)
		out.Close()
		if err != nil {
			h.renderMediaUpload(w, r, err.Error())
			return
		}
		webPath := "/assets/media/" + folder + "/" + safeName
		db, _ := sites.DB(r.Context())
		asset := models.MediaAsset{
			Name:   safeName,
			Path:   webPath,
			MIME:   header.Header.Get("Content-Type"),
			Size:   size,
			Folder: folder,
			Alt:    formString(r, "alt"),
		}
		if err := db.Create(&asset).Error; err != nil {
			h.renderMediaUpload(w, r, err.Error())
			return
		}
		redirectList(w, r, mediaBase)
		return
	}
	h.renderMediaUpload(w, r, "")
}

func (h *Handler) renderMediaUpload(w http.ResponseWriter, r *http.Request, errMsg string) {
	data := formData(map[string]any{
		"ActiveNav": "media",
		"BasePath":  mediaBase,
		"Subtitle":  "Upload a file to the media library.",
		"Title":     "Upload Media",
	})
	if errMsg != "" {
		data["Error"] = errMsg
	}
	h.render(w, r, "Upload Media", "admin/media_upload.html", data)
}

func (h *Handler) mediaDelete(w http.ResponseWriter, r *http.Request, idStr string) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	id, ok := parseID(idStr)
	if !ok {
		http.NotFound(w, r)
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
