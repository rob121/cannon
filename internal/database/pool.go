package database

import (
	"fmt"

	"github.com/rob121/cannon/internal/config"
	"gorm.io/gorm"
)

// ConfigurePool applies database-type-specific pool settings after gorm.Open.
func ConfigurePool(db *gorm.DB, cfg config.DatabaseConfig) error {
	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("database pool: %w", err)
	}
	if IsSQLite(cfg.Type) {
		sqlDB.SetMaxOpenConns(1)
		sqlDB.SetMaxIdleConns(1)
		return nil
	}
	sqlDB.SetMaxOpenConns(0)
	sqlDB.SetMaxIdleConns(0)
	return nil
}
