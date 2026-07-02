package extension

import (
	"fmt"
	"os"

	"github.com/rob121/cannon/internal/config"
	"github.com/rob121/cannon/internal/extdb"
	"gorm.io/gorm"
)

// OpenDB opens the current site database using Cannon connection rules.
// Requires --site= and CANNON_CONFIG (set automatically when Cannon starts the extension).
func OpenDB() (*gorm.DB, *config.SiteConfig, error) {
	return extdb.OpenFromFlags()
}

// SiteConfig loads the current site configuration from sites.json.
func SiteConfig() (*config.SiteConfig, error) {
	siteID := extdb.SiteIDFromArgs(os.Args)
	if siteID == "" {
		return nil, fmt.Errorf("missing --site argument")
	}
	return extdb.SiteConfig(siteID)
}

// SiteID returns the --site= value from process arguments.
func SiteID() string {
	return extdb.SiteIDFromArgs(os.Args)
}
