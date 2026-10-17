package settings_test

import (
	"context"
	"testing"

	"github.com/rob121/cannon/internal/config"
	"github.com/rob121/cannon/internal/database"
	"github.com/rob121/cannon/internal/settings"
	"github.com/rob121/cannon/internal/sites"
)

func TestDateFormatDefaultAndConfigured(t *testing.T) {
	site := &config.SiteConfig{
		ID: t.Name(),
		Database: config.DatabaseConfig{
			Type: "sqlite",
			DSN:  t.TempDir() + "/site.db",
		},
	}
	if err := database.Migrate(site); err != nil {
		t.Fatal(err)
	}
	ctx := sites.WithContext(context.Background(), site)

	got, err := settings.DateFormat(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if got != settings.DefaultDateFormat {
		t.Fatalf("default = %q", got)
	}

	store := settings.NewStore()
	if err := store.Save(ctx, settings.ScopeGlobal, settings.SectionGeneral, map[string]any{
		"date_format": "2006-01-02",
	}); err != nil {
		t.Fatal(err)
	}
	got, err = settings.DateFormat(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if got != "2006-01-02" {
		t.Fatalf("configured = %q", got)
	}
}

func TestValidDateFormat(t *testing.T) {
	if !settings.ValidDateFormat("01/02/2006") {
		t.Fatal("expected US layout valid")
	}
	if settings.ValidDateFormat("bogus") {
		t.Fatal("expected invalid layout rejected")
	}
}
