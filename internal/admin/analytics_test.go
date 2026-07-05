package admin

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/rob121/cannon/internal/config"
	"github.com/rob121/cannon/internal/database"
	"github.com/rob121/cannon/internal/models"
	"github.com/rob121/cannon/internal/realtime"
	"github.com/rob121/cannon/internal/sites"
)

func TestAnalyticsLiveEndpoint(t *testing.T) {
	hub, err := realtime.NewHub()
	if err != nil {
		t.Fatal(err)
	}
	defer hub.Shutdown(context.Background())

	site := &config.SiteConfig{
		ID: "analytics-test",
		Database: config.DatabaseConfig{
			Type: "sqlite",
			DSN:  t.TempDir() + "/site.db",
		},
	}
	if err := database.Migrate(site); err != nil {
		t.Fatal(err)
	}
	db, err := database.Get(site.ID)
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&models.Setting{
		Scope:   "global",
		Section: "analytics",
		Data:    `{"enabled":true}`,
	}).Error; err != nil {
		t.Fatal(err)
	}

	ctx := sites.WithContext(context.Background(), site)
	h := &Handler{realtime: hub}
	req := httptest.NewRequest(http.MethodGet, "/admin/analytics/live", nil).WithContext(ctx)
	rec := httptest.NewRecorder()
	h.analyticsLive(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	var stats realtime.Stats
	if err := json.NewDecoder(rec.Body).Decode(&stats); err != nil {
		t.Fatal(err)
	}
	if stats.UpdatedAt == "" {
		t.Fatalf("expected updated_at in payload: %+v", stats)
	}
}

func TestAnalyticsLiveDisabled(t *testing.T) {
	hub, err := realtime.NewHub()
	if err != nil {
		t.Fatal(err)
	}
	defer hub.Shutdown(context.Background())

	site := &config.SiteConfig{
		ID: "analytics-off",
		Database: config.DatabaseConfig{
			Type: "sqlite",
			DSN:  t.TempDir() + "/site.db",
		},
	}
	if err := database.Migrate(site); err != nil {
		t.Fatal(err)
	}

	ctx := sites.WithContext(context.Background(), site)
	h := &Handler{realtime: hub}
	req := httptest.NewRequest(http.MethodGet, "/admin/analytics/live", nil).WithContext(ctx)
	rec := httptest.NewRecorder()
	h.analyticsLive(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status=%d want 404", rec.Code)
	}
}
