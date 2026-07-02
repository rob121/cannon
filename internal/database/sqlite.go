package database

import (
	"fmt"
	"net/url"
	"path/filepath"
	"strings"
)

const (
	sqliteJournalMode = "WAL"
	sqliteBusyTimeout = "5000"
	sqliteForeignKeys = "on"
)

// IsSQLite reports whether the database config uses SQLite.
func IsSQLite(cfgType string) bool {
	t := strings.ToLower(strings.TrimSpace(cfgType))
	return t == "" || t == "sqlite"
}

// SQLiteDSN returns a GORM SQLite DSN with WAL, busy timeout, and foreign keys enabled.
func SQLiteDSN(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return path
	}
	if strings.HasPrefix(path, "file:") {
		return mergeSQLiteParams(path)
	}
	clean := filepath.ToSlash(path)
	return fmt.Sprintf("file:%s?_journal_mode=%s&_busy_timeout=%s&_foreign_keys=%s",
		clean, sqliteJournalMode, sqliteBusyTimeout, sqliteForeignKeys)
}

func mergeSQLiteParams(dsn string) string {
	u, err := url.Parse(dsn)
	if err != nil {
		return dsn
	}
	q := u.Query()
	setDefault(q, "_journal_mode", sqliteJournalMode)
	setDefault(q, "_busy_timeout", sqliteBusyTimeout)
	setDefault(q, "_foreign_keys", sqliteForeignKeys)
	u.RawQuery = q.Encode()
	return u.String()
}

func setDefault(q url.Values, key, value string) {
	if q.Get(key) == "" {
		q.Set(key, value)
	}
}
