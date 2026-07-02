package database

import (
	"fmt"

	"github.com/rob121/cannon/internal/config"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// Open creates a new GORM connection using Cannon's database rules.
// Extension processes should use this instead of opening SQLite directly.
func Open(cfg config.DatabaseConfig) (*gorm.DB, error) {
	dialector, err := dialectorFor(cfg)
	if err != nil {
		return nil, err
	}
	db, err := gorm.Open(dialector, &gorm.Config{
		Logger: logger.Default.LogMode(logger.Warn),
	})
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}
	if err := ConfigurePool(db, cfg); err != nil {
		return nil, err
	}
	return db, nil
}

// OpenSite opens the database for a site configuration.
func OpenSite(site *config.SiteConfig) (*gorm.DB, error) {
	if site == nil {
		return nil, fmt.Errorf("site is required")
	}
	return Open(site.Database)
}
