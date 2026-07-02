package admin

import (
	"net/http"

	"github.com/rob121/cannon/internal/models"
	"github.com/rob121/cannon/internal/sites"
)

const commentsBase = "/admin/comments"

type commentListRow struct {
	models.Comment
	ItemTitle string
}

func (h *Handler) comments(w http.ResponseWriter, r *http.Request, path string) {
	parts := pathParts("/comments", path)
	switch {
	case len(parts) == 0:
		h.commentList(w, r)
	case len(parts) == 2 && parts[1] == "approve":
		h.commentApprove(w, r, parts[0], true)
	case len(parts) == 2 && parts[1] == "unapprove":
		h.commentApprove(w, r, parts[0], false)
	case len(parts) == 2 && parts[1] == "delete":
		h.commentDelete(w, r, parts[0])
	default:
		http.NotFound(w, r)
	}
}

func (h *Handler) commentList(w http.ResponseWriter, r *http.Request) {
	db, _ := sites.DB(r.Context())
	page := parsePage(r)
	filter := r.URL.Query().Get("filter")
	q := db.Model(&models.Comment{})
	switch filter {
	case "approved":
		q = q.Where("approved = ?", true)
	case "pending":
		q = q.Where("approved = ?", false)
	}
	var total int64
	q.Count(&total)
	data := listPage(page, total, commentsBase,
		"Moderate item comments.",
		"", map[string]any{"ActiveNav": "comments"})
	order := applyListSort(r, data, map[string]string{"created": "created_at"}, "created")
	var rows []models.Comment
	q.Preload("Item").Preload("User").
		Offset((page - 1) * pageSize).Limit(pageSize).Order(order + " DESC").Find(&rows)
	listRows := make([]commentListRow, 0, len(rows))
	for _, row := range rows {
		lr := commentListRow{Comment: row}
		if row.Item != nil {
			lr.ItemTitle = row.Item.Title
		}
		listRows = append(listRows, lr)
	}
	data["Rows"] = listRows
	data["Filter"] = filter
	h.render(w, r, "Comments", "admin/comments.html", data)
}

func (h *Handler) commentApprove(w http.ResponseWriter, r *http.Request, idStr string, approved bool) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	id, ok := parseID(idStr)
	if !ok {
		http.NotFound(w, r)
		return
	}
	db, _ := sites.DB(r.Context())
	db.Model(&models.Comment{}).Where("comment_id = ?", id).Update("approved", approved)
	redirectList(w, r, commentsBase+listRedirectQuery(r))
}

func (h *Handler) commentDelete(w http.ResponseWriter, r *http.Request, idStr string) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	id, ok := parseID(idStr)
	if !ok {
		http.NotFound(w, r)
		return
	}
	db, _ := sites.DB(r.Context())
	if err := db.Delete(&models.Comment{}, id).Error; err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	redirectList(w, r, commentsBase+listRedirectQuery(r))
}
