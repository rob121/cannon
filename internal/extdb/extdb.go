// Package extdb helps Cannon extensions open the current site database safely.
package extdb

import (
	"fmt"
	"os"
	"strings"

	"github.com/rob121/cannon/internal/config"
	"github.com/rob121/cannon/internal/database"
	"gorm.io/gorm"
)

const configEnv = "CANNON_CONFIG"

// OpenSite loads sites.json, finds the site, and opens its database using Cannon's connection rules.
func OpenSite(siteID string) (*gorm.DB, *config.SiteConfig, error) {
	site, err := SiteConfig(siteID)
	if err != nil {
		return nil, nil, err
	}
	db, err := database.OpenSite(site)
	if err != nil {
		return nil, nil, err
	}
	return db, site, nil
}

// SiteConfig loads the site configuration without opening a database connection.
func SiteConfig(siteID string) (*config.SiteConfig, error) {
	app, err := loadApp()
	if err != nil {
		return nil, err
	}
	return config.FindSite(app, siteID)
}

// OpenFromFlags reads --site= from process args and opens the site database.
func OpenFromFlags() (*gorm.DB, *config.SiteConfig, error) {
	siteID := SiteIDFromArgs(os.Args)
	if siteID == "" {
		return nil, nil, fmt.Errorf("missing --site argument")
	}
	return OpenSite(siteID)
}

// SiteIDFromArgs returns the value of a --site= argument.
func SiteIDFromArgs(args []string) string {
	for _, arg := range args {
		if strings.HasPrefix(arg, "--site=") {
			return strings.TrimPrefix(arg, "--site=")
		}
	}
	return ""
}

func loadApp() (*config.App, error) {
	if path := os.Getenv(configEnv); path != "" {
		return config.LoadFile(path)
	}
	cfg, _, err := config.Load()
	if err != nil {
		return nil, err
	}
	if cfg == nil {
		return nil, fmt.Errorf("sites.json is not loaded")
	}
	return cfg, nil
}
