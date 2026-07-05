package admin

import (
	"net/http"

	"github.com/rob121/cannon/internal/content"
	"github.com/rob121/cannon/internal/models"
	"github.com/rob121/cannon/internal/sites"
	"github.com/rob121/cannon/internal/user"
)

const (
	trashBase  = "/admin/trash"
	reviewBase = "/admin/review"
)

func (h *Handler) itemTrash(w http.ResponseWriter, r *http.Request, path string) {
	parts := pathParts("/trash", path)
	if len(parts) == 1 && parts[0] == "empty" {
		h.itemTrashEmpty(w, r)
		return
	}
	if len(parts) == 2 && parts[1] == "restore" {
		h.itemTrashRestore(w, r, parts[0])
		return
	}
	h.itemTrashList(w, r)
}

func (h *Handler) itemTrashList(w http.ResponseWriter, r *http.Request) {
	db, _ := sites.DB(r.Context())
	page := parsePage(r)
	q := db.Model(&models.Item{}).Where("status = ?", models.ItemStatusTrashed)
	var total int64
	q.Count(&total)
	data := listPage(r, page, total, trashBase,
		"Restore or permanently delete trashed content items.",
		"", map[string]any{"ActiveNav": "trash"})
	var rows []models.Item
	db.Preload("Category").Preload("Author").
		Where("status = ?", models.ItemStatusTrashed).
		Order("updated_at desc").
		Offset((page - 1) * pageSizeFor(r)).Limit(pageSizeFor(r)).
		Find(&rows)
	listRows := make([]itemListRow, 0, len(rows))
	for _, row := range rows {
		lr := itemListRow{Item: row}
		if row.Category != nil {
			lr.CategoryName = row.Category.Name
		}
		if row.Author != nil {
			lr.AuthorName = row.Author.Username
		}
		listRows = append(listRows, lr)
	}
	data["Rows"] = listRows
	data["ListQuery"] = listQueryFromData(data)
	h.render(w, r, "Trash", "admin/trash.html", data)
}

func (h *Handler) itemTrashRestore(w http.ResponseWriter, r *http.Request, idStr string) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	id, ok := parseID(idStr)
	if !ok {
		h.notFound(w, r)
		return
	}
	db, _ := sites.DB(r.Context())
	if err := content.RestoreItem(r.Context(), db, id); err != nil {
		h.notFound(w, r)
		return
	}
	redirectList(w, r, trashBase+listRedirectQuery(r))
}

func (h *Handler) itemTrashEmpty(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	db, _ := sites.DB(r.Context())
	var ids []uint
	db.Model(&models.Item{}).Where("status = ?", models.ItemStatusTrashed).Pluck("item_id", &ids)
	for _, id := range ids {
		_ = content.DeleteItemPermanent(r.Context(), db, id)
	}
	redirectList(w, r, trashBase)
}

func (h *Handler) itemReview(w http.ResponseWriter, r *http.Request, path string) {
	parts := pathParts("/review", path)
	if len(parts) == 0 {
		if r.Method == http.MethodPost {
			h.itemReviewBulk(w, r)
			return
		}
		h.itemReviewList(w, r)
		return
	}
	if len(parts) == 2 {
		switch parts[1] {
		case "approve":
			h.itemReviewApprove(w, r, parts[0])
		case "reject":
			h.itemReviewReject(w, r, parts[0])
		}
	}
}

func (h *Handler) itemReviewList(w http.ResponseWriter, r *http.Request) {
	db, _ := sites.DB(r.Context())
	page := parsePage(r)
	q := db.Model(&models.Item{}).Where("status = ?", models.ItemStatusPending)
	var total int64
	q.Count(&total)
	data := listPage(r, page, total, reviewBase,
		"Items submitted for review awaiting publication approval.",
		"", map[string]any{"ActiveNav": "review"})
	var rows []models.Item
	db.Preload("Category").Preload("Author").
		Where("status = ?", models.ItemStatusPending).
		Order("updated_at desc").
		Offset((page - 1) * pageSizeFor(r)).Limit(pageSizeFor(r)).
		Find(&rows)
	listRows := make([]itemListRow, 0, len(rows))
	for _, row := range rows {
		lr := itemListRow{Item: row}
		if row.Category != nil {
			lr.CategoryName = row.Category.Name
		}
		if row.Author != nil {
			lr.AuthorName = row.Author.Username
		}
		listRows = append(listRows, lr)
	}
	data["Rows"] = listRows
	data["ListQuery"] = listQueryFromData(data)
	h.render(w, r, "Review Queue", "admin/review.html", data)
}

func (h *Handler) itemReviewBulk(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	action := formString(r, "bulk_action")
	db, _ := sites.DB(r.Context())
	for _, idStr := range r.Form["item_ids"] {
		id, ok := parseID(idStr)
		if !ok {
			continue
		}
		switch action {
		case "approve":
			db.Model(&models.Item{}).Where("item_id = ? AND status = ?", id, models.ItemStatusPending).
				Update("status", models.ItemStatusPublished)
		case "reject":
			db.Model(&models.Item{}).Where("item_id = ? AND status = ?", id, models.ItemStatusPending).
				Update("status", models.ItemStatusDraft)
		}
	}
	redirectList(w, r, reviewBase+listRedirectQuery(r))
}

func (h *Handler) itemReviewApprove(w http.ResponseWriter, r *http.Request, idStr string) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	id, ok := parseID(idStr)
	if !ok {
		h.notFound(w, r)
		return
	}
	db, _ := sites.DB(r.Context())
	db.Model(&models.Item{}).Where("item_id = ? AND status = ?", id, models.ItemStatusPending).
		Update("status", models.ItemStatusPublished)
	redirectList(w, r, reviewBase+listRedirectQuery(r))
}

func (h *Handler) itemReviewReject(w http.ResponseWriter, r *http.Request, idStr string) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	id, ok := parseID(idStr)
	if !ok {
		h.notFound(w, r)
		return
	}
	db, _ := sites.DB(r.Context())
	db.Model(&models.Item{}).Where("item_id = ? AND status = ?", id, models.ItemStatusPending).
		Update("status", models.ItemStatusDraft)
	redirectList(w, r, reviewBase+listRedirectQuery(r))
}

func (h *Handler) itemRevisions(w http.ResponseWriter, r *http.Request, idStr string) {
	id, ok := parseID(idStr)
	if !ok {
		h.notFound(w, r)
		return
	}
	db, _ := sites.DB(r.Context())
	var row models.Item
	if err := db.First(&row, id).Error; err != nil {
		h.notFound(w, r, "This item could not be found.")
		return
	}
	revisions, err := content.ListItemRevisions(r.Context(), id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	var compareDiffs []content.RevisionDiff
	compareLabel := ""
	if len(revisions) >= 2 {
		a, _ := content.LoadRevisionSnapshot(revisions[0])
		b, _ := content.LoadRevisionSnapshot(revisions[1])
		compareDiffs = content.CompareRevisionSnapshots(b, a)
		compareLabel = content.RevisionLabel(revisions[1]) + " → " + content.RevisionLabel(revisions[0])
	}
	data := formData(map[string]any{
		"ActiveNav":    "items",
		"Row":          row,
		"Revisions":    revisions,
		"CompareDiffs": compareDiffs,
		"CompareLabel": compareLabel,
		"BasePath":     itemsBase,
	})
	h.render(w, r, "Item Revisions", "admin/items_revisions.html", data)
}

func (h *Handler) itemRevisionRestore(w http.ResponseWriter, r *http.Request, idStr, revStr string) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	id, ok := parseID(idStr)
	if !ok {
		h.notFound(w, r)
		return
	}
	revID, ok := parseID(revStr)
	if !ok {
		h.notFound(w, r)
		return
	}
	editorID, editorName := currentEditor(r)
	if err := content.FireRevisionRestore(r.Context(), id, revID); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := content.RollbackItemRevision(r.Context(), id, revID, editorID, editorName); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	redirectList(w, r, itemsBase+"/"+idStr+"/revisions")
}

func (h *Handler) itemPreviewToken(w http.ResponseWriter, r *http.Request, idStr string) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	id, ok := parseID(idStr)
	if !ok {
		h.notFound(w, r)
		return
	}
	if _, err := content.EnsurePreviewToken(r.Context(), id); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	redirectList(w, r, itemsBase+"/"+idStr+"?preview=1#preview")
}

func (h *Handler) itemSubmitReview(w http.ResponseWriter, r *http.Request, idStr string) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	id, ok := parseID(idStr)
	if !ok {
		h.notFound(w, r)
		return
	}
	db, _ := sites.DB(r.Context())
	db.Model(&models.Item{}).Where("item_id = ?", id).
		Update("status", models.ItemStatusPending)
	redirectList(w, r, itemsBase+"/"+idStr)
}

func currentEditor(r *http.Request) (*uint, string) {
	svc, err := user.FromContext(r.Context())
	if err != nil {
		return nil, ""
	}
	u, err := svc.Current(r.Context())
	if err != nil {
		return nil, ""
	}
	id := u.UserID
	name := u.Username
	return &id, name
}
