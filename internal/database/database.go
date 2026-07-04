package database

import (
	"fmt"
	"sync"

	"github.com/rob121/cannon/internal/config"
	"github.com/rob121/cannon/internal/models"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

var (
	mu    sync.RWMutex
	pools = map[string]*gorm.DB{}
)

// Connect opens a GORM connection for a site and caches it.
func Connect(site *config.SiteConfig) (*gorm.DB, error) {
	mu.RLock()
	if db, ok := pools[site.ID]; ok {
		mu.RUnlock()
		return db, nil
	}
	mu.RUnlock()

	db, err := OpenSite(site)
	if err != nil {
		return nil, fmt.Errorf("open database for site %s: %w", site.ID, err)
	}

	mu.Lock()
	pools[site.ID] = db
	mu.Unlock()

	return db, nil
}

// Get returns a cached connection for a site id.
func Get(siteID string) (*gorm.DB, error) {
	mu.RLock()
	defer mu.RUnlock()
	db, ok := pools[siteID]
	if !ok {
		return nil, fmt.Errorf("database not connected for site %s", siteID)
	}
	return db, nil
}

// Migrate runs GORM auto-migration for a site database.
func Migrate(site *config.SiteConfig) error {
	db, err := Connect(site)
	if err != nil {
		return err
	}
	if err := RepairMenuSchema(db); err != nil {
		return err
	}
	if err := RepairRoutesSchema(db); err != nil {
		return err
	}
	for _, model := range models.All() {
		if err := db.AutoMigrate(model); err != nil {
			return err
		}
	}
	if err := RepairContentLocaleSchema(db); err != nil {
		return err
	}
	return nil
}

// CloseAll closes all cached database connections.
func CloseAll() {
	mu.Lock()
	defer mu.Unlock()
	for id, db := range pools {
		if sqlDB, err := db.DB(); err == nil {
			_ = sqlDB.Close()
		}
		delete(pools, id)
	}
}

func dialectorFor(cfg config.DatabaseConfig) (gorm.Dialector, error) {
	switch cfg.Type {
	case "sqlite", "":
		path := cfg.DSN
		if path == "" {
			return nil, fmt.Errorf("sqlite dsn is required")
		}
		return sqlite.Open(SQLiteDSN(path)), nil
	case "mysql":
		return mysql.Open(cfg.DSN), nil
	case "postgres":
		return postgres.Open(cfg.DSN), nil
	default:
		return nil, fmt.Errorf("unsupported database type: %s", cfg.Type)
	}
}
