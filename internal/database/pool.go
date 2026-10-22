package database

import (
	"fmt"
	"time"

	"github.com/rob121/cannon/internal/config"
	"gorm.io/gorm"
)

// SQLiteMaxOpenConns is the per-process pool for one WAL database.
// SQLite still allows a single writer. Extra connections let concurrent
// requests read at the same time instead of queueing on one handle.
// The cap stays small so Cannon and extension processes do not open a
// large pile of connections against the same file.
const SQLiteMaxOpenConns = 4

// sqliteMaxIdleConns keeps a couple of handles warm without pinning the
// whole pool in every idle extension process.
const sqliteMaxIdleConns = 2

// sqliteConnMaxIdleTime drops unused connections so an extension that
// opened the site database and went quiet does not hold them open.
const sqliteConnMaxIdleTime = 2 * time.Minute

// ConfigurePool applies database-type-specific pool settings after gorm.Open.
func ConfigurePool(db *gorm.DB, cfg config.DatabaseConfig) error {
	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("database pool: %w", err)
	}
	if IsSQLite(cfg.Type) {
		sqlDB.SetMaxOpenConns(SQLiteMaxOpenConns)
		sqlDB.SetMaxIdleConns(sqliteMaxIdleConns)
		sqlDB.SetConnMaxIdleTime(sqliteConnMaxIdleTime)
		return nil
	}
	sqlDB.SetMaxOpenConns(0)
	sqlDB.SetMaxIdleConns(0)
	return nil
}
