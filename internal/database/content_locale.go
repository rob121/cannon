package database

import (
	"strings"

	"gorm.io/gorm"
)

const defaultContentLocale = "en-US"

// RepairContentLocaleSchema backfills locale columns and replaces slug-only unique indexes.
func RepairContentLocaleSchema(db *gorm.DB) error {
	if db == nil {
		return nil
	}
	_ = db.Exec(`UPDATE items SET locale = ? WHERE locale IS NULL OR locale = ''`, defaultContentLocale).Error
	_ = db.Exec(`UPDATE categories SET locale = ? WHERE locale IS NULL OR locale = ''`, defaultContentLocale).Error
	if db.Dialector.Name() == "sqlite" {
		if err := dropSlugOnlyUniqueIndexes(db, "items"); err != nil {
			return err
		}
		if err := dropSlugOnlyUniqueIndexes(db, "categories"); err != nil {
			return err
		}
	}
	return MigrateSearchIndex(db)
}

// MigrateSearchIndex creates the SQLite FTS table when supported.
func MigrateSearchIndex(db *gorm.DB) error {
	if db == nil || db.Dialector.Name() != "sqlite" {
		return nil
	}
	err := db.Exec(`CREATE VIRTUAL TABLE IF NOT EXISTS item_search_fts USING fts5(
		item_id UNINDEXED,
		locale UNINDEXED,
		title,
		intro,
		body,
		fields,
		tokenize='porter unicode61'
	)`).Error
	if err != nil && ftsUnavailable(err) {
		return nil
	}
	return err
}

func ftsUnavailable(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "no such module: fts5") || strings.Contains(msg, "unknown tokenizer")
}

func dropSlugOnlyUniqueIndexes(db *gorm.DB, table string) error {
	type indexRow struct {
		Name string
		SQL  string
	}
	var rows []indexRow
	if err := db.Raw(`SELECT name, sql FROM sqlite_master WHERE type = 'index' AND tbl_name = ?`, table).Scan(&rows).Error; err != nil {
		return err
	}
	for _, row := range rows {
		if row.Name == "" || strings.Contains(row.Name, "locale_slug") {
			continue
		}
		lower := strings.ToLower(row.SQL)
		if strings.Contains(lower, "slug") && !strings.Contains(lower, "locale") {
			if err := db.Exec(`DROP INDEX IF EXISTS ` + row.Name).Error; err != nil {
				return err
			}
		}
	}
	return nil
}
