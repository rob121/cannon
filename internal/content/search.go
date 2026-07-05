package content

import (
	"context"
	"fmt"
	"strings"

	"github.com/rob121/cannon/internal/models"
	"github.com/rob121/cannon/internal/sites"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// SearchIndexAvailable reports whether the SQLite FTS index is available.
func SearchIndexAvailable(db *gorm.DB) bool {
	if db == nil || db.Dialector.Name() != "sqlite" {
		return false
	}
	var name string
	err := db.Raw(`SELECT name FROM sqlite_master WHERE type = 'table' AND name = 'item_search_fts'`).Scan(&name).Error
	return err == nil && name == "item_search_fts"
}

// RebuildSearchIndex rebuilds the FTS table from all items.
func RebuildSearchIndex(ctx context.Context, db *gorm.DB) error {
	if !SearchIndexAvailable(db) {
		if err := ensureSearchIndex(db); err != nil {
			return err
		}
	}
	if err := db.Exec(`DELETE FROM item_search_fts`).Error; err != nil {
		return err
	}
	var items []models.Item
	if err := db.Find(&items).Error; err != nil {
		return err
	}
	for i := range items {
		if err := UpsertSearchIndex(ctx, db, &items[i]); err != nil {
			return err
		}
	}
	return nil
}

// UpsertSearchIndex inserts or replaces a single item in the FTS table.
func UpsertSearchIndex(ctx context.Context, db *gorm.DB, item *models.Item) error {
	if item == nil || item.ItemID == 0 {
		return nil
	}
	if !SearchIndexAvailable(db) {
		if err := ensureSearchIndex(db); err != nil {
			return err
		}
	}
	if !SearchIndexAvailable(db) {
		return nil
	}
	fieldsText, err := searchFieldsText(ctx, db, item.ItemID)
	if err != nil {
		return err
	}
	locale := strings.TrimSpace(item.Locale)
	if locale == "" {
		locale = LocaleFromContext(ctx)
	}
	if err := db.Exec(`DELETE FROM item_search_fts WHERE item_id = ?`, item.ItemID).Error; err != nil {
		return err
	}
	return db.Exec(
		`INSERT INTO item_search_fts (item_id, locale, title, intro, body, fields) VALUES (?, ?, ?, ?, ?, ?)`,
		item.ItemID, locale, item.Title, item.Intro, item.Body, fieldsText,
	).Error
}

// RemoveSearchIndex deletes an item from the FTS table.
func RemoveSearchIndex(db *gorm.DB, itemID uint) {
	if itemID == 0 || !SearchIndexAvailable(db) {
		return
	}
	_ = db.Exec(`DELETE FROM item_search_fts WHERE item_id = ?`, itemID).Error
}

func ensureSearchIndex(db *gorm.DB) error {
	if db == nil || db.Dialector.Name() != "sqlite" {
		return nil
	}
	err := sqliteSilentExec(db, `CREATE VIRTUAL TABLE IF NOT EXISTS item_search_fts USING fts5(
		item_id UNINDEXED,
		locale UNINDEXED,
		title,
		intro,
		body,
		fields,
		tokenize='porter unicode61'
	)`)
	if err != nil {
		if sqliteFTSUnavailable(err) {
			return nil
		}
		return err
	}
	return nil
}

func sqliteFTSUnavailable(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "no such module: fts5") || strings.Contains(msg, "unknown tokenizer")
}

func sqliteSilentExec(db *gorm.DB, sql string, args ...any) error {
	if db == nil {
		return nil
	}
	return db.Session(&gorm.Session{Logger: logger.Discard}).Exec(sql, args...).Error
}

func searchFieldsText(ctx context.Context, db *gorm.DB, itemID uint) (string, error) {
	values, err := ItemFieldMap(ctx, itemID)
	if err != nil {
		return "", err
	}
	if len(values) == 0 {
		return "", nil
	}
	parts := make([]string, 0, len(values))
	for _, v := range values {
		v = strings.TrimSpace(v)
		if v != "" {
			parts = append(parts, v)
		}
	}
	return strings.Join(parts, "\n"), nil
}

func applySearchQuery(ctx context.Context, db *gorm.DB, q *gorm.DB, opts ListOptions) (*gorm.DB, error) {
	query := strings.TrimSpace(opts.Query)
	if query == "" {
		return q, nil
	}
	if SearchIndexAvailable(db) {
		ids, err := ftsItemIDs(db, query, LocaleFromContext(ctx))
		if err == nil && len(ids) > 0 {
			order := ftsOrderClause(ids)
			return q.Where("items.item_id IN ?", ids).Order(order), nil
		}
	}
	like := "%" + query + "%"
	return q.Where("items.title LIKE ? OR items.intro LIKE ? OR items.body LIKE ?", like, like, like), nil
}

func ftsItemIDs(db *gorm.DB, query, locale string) ([]uint, error) {
	match := ftsMatchQuery(query)
	if match == "" {
		return nil, nil
	}
	type row struct {
		ItemID uint
	}
	var rows []row
	err := db.Raw(`
		SELECT item_id FROM item_search_fts
		WHERE item_search_fts MATCH ? AND locale = ?
		ORDER BY bm25(item_search_fts)
		LIMIT 500
	`, match, locale).Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	ids := make([]uint, 0, len(rows))
	for _, row := range rows {
		ids = append(ids, row.ItemID)
	}
	return ids, nil
}

func ftsOrderClause(ids []uint) string {
	if len(ids) == 0 {
		return "created_at DESC"
	}
	parts := make([]string, 0, len(ids))
	for i, id := range ids {
		parts = append(parts, fmt.Sprintf("WHEN %d THEN %d", id, i))
	}
	return "CASE items.item_id " + strings.Join(parts, " ") + " END"
}

func ftsMatchQuery(query string) string {
	terms := strings.Fields(query)
	if len(terms) == 0 {
		return ""
	}
	parts := make([]string, 0, len(terms))
	for _, term := range terms {
		term = strings.TrimSpace(strings.ReplaceAll(term, `"`, ""))
		if term == "" {
			continue
		}
		parts = append(parts, `"`+term+`"`)
	}
	if len(parts) == 0 {
		return ""
	}
	return strings.Join(parts, " AND ")
}

func applyFieldFilters(db *gorm.DB, q *gorm.DB, filters map[string]string) *gorm.DB {
	if len(filters) == 0 {
		return q
	}
	idx := 0
	for name, value := range filters {
		name = strings.TrimSpace(name)
		value = strings.TrimSpace(value)
		if name == "" || value == "" {
			continue
		}
		fvAlias := fmt.Sprintf("fv_filter_%d", idx)
		cfAlias := fmt.Sprintf("cf_filter_%d", idx)
		q = q.Joins(fmt.Sprintf(
			`JOIN item_field_values %s ON %s.item_id = items.item_id`, fvAlias, fvAlias,
		)).Joins(fmt.Sprintf(
			`JOIN content_fields %s ON %s.field_id = %s.field_id AND %s.name = ?`, cfAlias, cfAlias, fvAlias, cfAlias,
		), name).Where(fvAlias+`.value LIKE ?`, "%"+value+"%")
		idx++
	}
	return q
}

// ParseFieldFilters extracts cf_<name> query parameters.
func ParseFieldFilters(values map[string][]string) map[string]string {
	out := map[string]string{}
	for key, vals := range values {
		if !strings.HasPrefix(key, "cf_") || len(vals) == 0 {
			continue
		}
		name := strings.TrimSpace(strings.TrimPrefix(key, "cf_"))
		value := strings.TrimSpace(vals[0])
		if name != "" && value != "" {
			out[name] = value
		}
	}
	return out
}

// SearchableFields returns active text-like custom fields for search filters.
func SearchableFields(ctx context.Context) ([]models.ContentField, error) {
	db, err := sites.DB(ctx)
	if err != nil {
		return nil, err
	}
	var fields []models.ContentField
	err = db.Where("status = ? AND type IN ?", models.StatusActive, []string{"text", "textarea", "select", "url", "email"}).
		Order("label ASC, name ASC").Find(&fields).Error
	return fields, err
}
