package extdb

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/rob121/cannon/internal/config"
	"github.com/rob121/cannon/internal/database"
)

func TestOpenSiteSQLiteUsesSharedRules(t *testing.T) {
	dir := t.TempDir()
	dbPath := dir + "/example.sqlite"
	cfgPath := dir + "/sites.json"
	app := &config.App{
		DataRoot: dir,
		Sites: []config.SiteConfig{{
			ID:   "example",
			Name: "Example",
			Database: config.DatabaseConfig{
				Type: "sqlite",
				DSN:  dbPath,
			},
		}},
	}
	raw, err := json.Marshal(app)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(cfgPath, raw, 0644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("CANNON_CONFIG", cfgPath)

	db, site, err := OpenSite("example")
	if err != nil {
		t.Fatal(err)
	}
	if site.ID != "example" {
		t.Fatalf("unexpected site: %#v", site)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	if sqlDB.Stats().MaxOpenConnections != 1 {
		t.Fatalf("expected sqlite max open conns 1")
	}
	if !database.IsSQLite(site.Database.Type) {
		t.Fatal("expected sqlite site")
	}
}

func TestSiteIDFromArgs(t *testing.T) {
	args := []string{"ext", "--site=example", "--socket=/tmp/x.sock"}
	if got := SiteIDFromArgs(args); got != "example" {
		t.Fatalf("expected example, got %q", got)
	}
}
