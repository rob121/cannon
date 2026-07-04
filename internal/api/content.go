package api

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	cms "github.com/rob121/cannon/internal/content"
	"github.com/rob121/cannon/internal/captcha"
	"github.com/rob121/cannon/internal/groups"
	"github.com/rob121/cannon/internal/models"
	"github.com/rob121/cannon/internal/sites"
	"gorm.io/gorm"
)

func (h *Handler) serveItems(w http.ResponseWriter, r *http.Request, parts []string) {
	ctx := r.Context()
	viewerGroups, _ := ViewerGroups(ctx)
	if len(viewerGroups) == 0 {
		viewerGroups, _ = resolveViewerGroups(ctx)
	}
	if len(parts) == 0 {
		if r.Method != http.MethodGet {
			methodNotAllowed(w)
			return
		}
		h.listItems(w, r, viewerGroups)
		return
	}
	if parts[0] == "by-slug" && len(parts) >= 2 {
		if r.Method != http.MethodGet {
			methodNotAllowed(w)
			return
		}
		h.getItemBySlug(w, r, parts[1], viewerGroups)
		return
	}
	id, err := strconv.ParseUint(parts[0], 10, 64)
	if err != nil || id == 0 {
		writeError(w, http.StatusNotFound, "not_found", "Item not found")
		return
	}
	if len(parts) == 2 && parts[1] == "comments" {
		h.serveItemComments(w, r, uint(id), viewerGroups)
		return
	}
	if len(parts) == 2 && parts[1] == "translations" {
		if r.Method != http.MethodGet {
			methodNotAllowed(w)
			return
		}
		h.itemTranslations(w, r, uint(id), viewerGroups)
		return
	}
	if len(parts) != 1 || r.Method != http.MethodGet {
		writeError(w, http.StatusNotFound, "not_found", "Item not found")
		return
	}
	h.getItemByID(w, r, uint(id), viewerGroups)
}

func (h *Handler) listItems(w http.ResponseWriter, r *http.Request, viewerGroups []uint) {
	ctx := r.Context()
	page, pageSize := parsePageQuery(r)
	opts := cms.ListOptions{Page: page, Limit: pageSize}
	if v := strings.TrimSpace(r.URL.Query().Get("q")); v != "" {
		opts.Query = v
	}
	if v := strings.TrimSpace(r.URL.Query().Get("category_id")); v != "" {
		if n, err := strconv.ParseUint(v, 10, 64); err == nil {
			opts.CategoryID = uint(n)
		}
	}
	if v := strings.TrimSpace(r.URL.Query().Get("tag_id")); v != "" {
		if n, err := strconv.ParseUint(v, 10, 64); err == nil {
			opts.TagID = uint(n)
		}
	}
	if r.URL.Query().Get("featured") == "1" {
		opts.Featured = true
	}
	items, total, err := cms.ListItems(ctx, viewerGroups, opts)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "server_error", err.Error())
		return
	}
	data := make([]itemJSON, 0, len(items))
	for i := range items {
		row, err := itemToJSON(ctx, &items[i])
		if err != nil {
			continue
		}
		data = append(data, row)
	}
	writeJSON(w, http.StatusOK, ListResponse{Data: data, Meta: PageMeta{Page: page, PageSize: pageSize, Total: total}})
}

func (h *Handler) getItemByID(w http.ResponseWriter, r *http.Request, id uint, viewerGroups []uint) {
	item, err := loadVisibleItemByID(r.Context(), id, viewerGroups)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			writeError(w, http.StatusNotFound, "not_found", "Item not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "server_error", err.Error())
		return
	}
	body, err := itemToJSON(r.Context(), item)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "server_error", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, body)
}

func (h *Handler) getItemBySlug(w http.ResponseWriter, r *http.Request, slug string, viewerGroups []uint) {
	item, err := cms.ItemBySlug(r.Context(), slug, viewerGroups)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			writeError(w, http.StatusNotFound, "not_found", "Item not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "server_error", err.Error())
		return
	}
	body, err := itemToJSON(r.Context(), item)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "server_error", err.Error())
		return
	}
	writeJSON(w, http.StatusOK, body)
}

func (h *Handler) itemTranslations(w http.ResponseWriter, r *http.Request, id uint, viewerGroups []uint) {
	ctx := r.Context()
	item, err := loadVisibleItemByID(ctx, id, viewerGroups)
	if err != nil {
		writeError(w, http.StatusNotFound, "not_found", "Item not found")
		return
	}
	if item.TranslationGroupID == nil || *item.TranslationGroupID == 0 {
		writeJSON(w, http.StatusOK, map[string]any{"data": []any{}})
		return
	}
	q, err := cms.VisibleItemsQuery(ctx, viewerGroups)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "server_error", err.Error())
		return
	}
	var siblings []models.Item
	if err := q.Where("translation_group_id = ? AND item_id <> ?", *item.TranslationGroupID, item.ItemID).Find(&siblings).Error; err != nil {
		writeError(w, http.StatusInternalServerError, "server_error", err.Error())
		return
	}
	out := make([]map[string]any, 0, len(siblings))
	for i := range siblings {
		out = append(out, map[string]any{
			"item_id": siblings[i].ItemID, "locale": siblings[i].Locale,
			"slug": siblings[i].Slug, "title": siblings[i].Title,
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": out})
}

func (h *Handler) serveItemComments(w http.ResponseWriter, r *http.Request, itemID uint, viewerGroups []uint) {
	ctx := r.Context()
	if _, err := loadVisibleItemByID(ctx, itemID, viewerGroups); err != nil {
		writeError(w, http.StatusNotFound, "not_found", "Item not found")
		return
	}
	switch r.Method {
	case http.MethodGet:
		comments, err := cms.ApprovedComments(ctx, itemID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "server_error", err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"data": comments})
	case http.MethodPost:
		h.postItemComment(w, r, itemID)
	default:
		methodNotAllowed(w)
	}
}

type commentPostBody struct {
	Body         string `json:"body"`
	AuthorName   string `json:"author_name"`
	AuthorEmail  string `json:"author_email"`
	CaptchaToken string `json:"captcha_token"`
}

func (h *Handler) postItemComment(w http.ResponseWriter, r *http.Request, itemID uint) {
	ctx := r.Context()
	settings, err := cms.LoadSettings(ctx)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "server_error", err.Error())
		return
	}
	if !settings.ShowComments || !settings.AllowComments {
		writeError(w, http.StatusForbidden, "forbidden", "Comments are disabled")
		return
	}
	var req commentPostBody
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", "Invalid JSON body")
		return
	}
	if err := captcha.VerifyJSON(ctx, r, captcha.CaptchaContextComment, req.CaptchaToken); err != nil {
		writeError(w, http.StatusBadRequest, "captcha_failed", captcha.UserFacingError(err))
		return
	}
	var userID *uint
	authenticated := false
	if id, ok := UserID(ctx); ok {
		authenticated = true
		can, err := cms.CanCreateComment(ctx, id)
		if err != nil || !can {
			writeError(w, http.StatusForbidden, "forbidden", "Permission denied")
			return
		}
		userID = &id
	}
	in := cms.CommentInput{
		ItemID: itemID, UserID: userID, Body: strings.TrimSpace(req.Body),
		AuthorName: req.AuthorName, AuthorEmail: req.AuthorEmail, IP: clientIP(r),
	}
	if _, err := cms.CreateComment(ctx, in, authenticated); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", "Could not post comment")
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"posted": true})
}

func (h *Handler) serveCategories(w http.ResponseWriter, r *http.Request, parts []string) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}
	ctx := r.Context()
	viewerGroups, _ := resolveViewerGroups(ctx)
	if len(parts) == 0 {
		rows, err := cms.CategoryTree(ctx)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "server_error", err.Error())
			return
		}
		filtered := filterVisibleCategories(rows, viewerGroups)
		writeJSON(w, http.StatusOK, map[string]any{"data": categoryRowsJSON(filtered)})
		return
	}
	id, err := strconv.ParseUint(parts[0], 10, 64)
	if err != nil || id == 0 {
		writeError(w, http.StatusNotFound, "not_found", "Category not found")
		return
	}
	db, err := sites.DB(ctx)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "server_error", err.Error())
		return
	}
	var cat models.Category
	if err := db.Preload("Groups").First(&cat, id).Error; err != nil {
		writeError(w, http.StatusNotFound, "not_found", "Category not found")
		return
	}
	if !groups.CanViewContent(viewerGroups, cat.Groups) {
		writeError(w, http.StatusNotFound, "not_found", "Category not found")
		return
	}
	writeJSON(w, http.StatusOK, categoryRowJSON(cat))
}

func filterVisibleCategories(rows []models.Category, viewerGroups []uint) []models.Category {
	var out []models.Category
	for _, c := range rows {
		if groups.CanViewContent(viewerGroups, c.Groups) {
			out = append(out, c)
		}
	}
	return out
}

func categoryRowsJSON(rows []models.Category) []map[string]any {
	out := make([]map[string]any, 0, len(rows))
	for _, c := range rows {
		out = append(out, categoryRowJSON(c))
	}
	return out
}

func categoryRowJSON(c models.Category) map[string]any {
	return map[string]any{
		"category_id": c.CategoryID, "name": c.Name, "slug": c.Slug,
		"parent_id": c.ParentID, "locale": c.Locale,
	}
}

func (h *Handler) serveTags(w http.ResponseWriter, r *http.Request, parts []string) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}
	ctx := r.Context()
	if len(parts) == 0 {
		tags, err := cms.ListTags(ctx)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "server_error", err.Error())
			return
		}
		out := make([]tagJSON, 0, len(tags))
		for _, t := range tags {
			out = append(out, tagJSON{TagID: t.TagID, Name: t.Name, Slug: t.Slug})
		}
		writeJSON(w, http.StatusOK, map[string]any{"data": out})
		return
	}
	id, err := strconv.ParseUint(parts[0], 10, 64)
	if err != nil || id == 0 {
		writeError(w, http.StatusNotFound, "not_found", "Tag not found")
		return
	}
	db, err := sites.DB(ctx)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "server_error", err.Error())
		return
	}
	var tag models.Tag
	if err := db.First(&tag, id).Error; err != nil {
		writeError(w, http.StatusNotFound, "not_found", "Tag not found")
		return
	}
	writeJSON(w, http.StatusOK, tagJSON{TagID: tag.TagID, Name: tag.Name, Slug: tag.Slug})
}

func (h *Handler) serveSearch(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		methodNotAllowed(w)
		return
	}
	ctx := r.Context()
	viewerGroups, _ := resolveViewerGroups(ctx)
	page, pageSize := parsePageQuery(r)
	opts := cms.ListOptions{
		Page: page, Limit: pageSize,
		Query: strings.TrimSpace(r.URL.Query().Get("q")),
	}
	items, total, err := cms.ListItems(ctx, viewerGroups, opts)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "server_error", err.Error())
		return
	}
	data := make([]itemJSON, 0, len(items))
	for i := range items {
		row, _ := itemToJSON(ctx, &items[i])
		data = append(data, row)
	}
	writeJSON(w, http.StatusOK, ListResponse{Data: data, Meta: PageMeta{Page: page, PageSize: pageSize, Total: total}})
}

func (h *Handler) serveMedia(w http.ResponseWriter, r *http.Request, parts []string) {
	if r.Method != http.MethodGet || len(parts) != 1 {
		writeError(w, http.StatusNotFound, "not_found", "Media not found")
		return
	}
	id, err := strconv.ParseUint(parts[0], 10, 64)
	if err != nil || id == 0 {
		writeError(w, http.StatusNotFound, "not_found", "Media not found")
		return
	}
	ctx := r.Context()
	viewerGroups, _ := resolveViewerGroups(ctx)
	db, err := sites.DB(ctx)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "server_error", err.Error())
		return
	}
	var media models.MediaAsset
	if err := db.First(&media, id).Error; err != nil {
		writeError(w, http.StatusNotFound, "not_found", "Media not found")
		return
	}
	visible, err := mediaVisibleOnItems(ctx, &media, viewerGroups)
	if err != nil || !visible {
		writeError(w, http.StatusNotFound, "not_found", "Media not found")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"media_id": media.MediaID, "name": media.Name, "path": media.Path,
		"mime": media.MIME, "size": media.Size, "alt": media.Alt,
		"width": media.Width, "height": media.Height,
	})
}

func (h *Handler) serveAuthors(w http.ResponseWriter, r *http.Request, parts []string) {
	if r.Method != http.MethodGet || len(parts) != 1 {
		writeError(w, http.StatusNotFound, "not_found", "Author not found")
		return
	}
	id, err := strconv.ParseUint(parts[0], 10, 64)
	if err != nil || id == 0 {
		writeError(w, http.StatusNotFound, "not_found", "Author not found")
		return
	}
	profile, err := cms.LoadAuthorProfile(r.Context(), uint(id))
	if err != nil || profile == nil {
		writeError(w, http.StatusNotFound, "not_found", "Author not found")
		return
	}
	fields := make([]map[string]any, 0, len(profile.Fields))
	for _, f := range profile.Fields {
		fields = append(fields, map[string]any{"name": f.Name, "type": f.Type, "value": f.Value})
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"display_name": profile.DisplayName, "email": profile.Email,
		"avatar_url": profile.AvatarURL, "fields": fields,
	})
}
