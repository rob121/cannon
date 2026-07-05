package templateengine

import (
	"html/template"
	"strings"
	"testing"

	"github.com/rob121/cannon/internal/config"
	"github.com/rob121/cannon/internal/realtime"
	"github.com/rob121/cannon/internal/themes"
)

func TestDashboardRealtimeConfigNotEscaped(t *testing.T) {
	raw, err := realtime.ConfigJSON("example", "http", "127.0.0.1:8001", true)
	if err != nil {
		t.Fatal(err)
	}
	e := New(&config.SiteConfig{ID: "example"}, themes.Selection{}, nil, nil, testAdminFuncs())
	tmpl, err := e.parse("admin/dashboard.html")
	if err != nil {
		t.Fatal(err)
	}
	var buf strings.Builder
	err = tmpl.Execute(&buf, map[string]any{
		"RealtimeEnabled":       true,
		"RealtimeConfigJSON":    template.JS(raw),
		"VisitorsOnline":        2,
		"VisitorsAuthenticated": 1,
		"AnalyticsPages":        []realtime.PageStat{{Path: "/", Count: 2}},
	})
	if err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if strings.Contains(out, "&#34;") {
		t.Fatalf("dashboard escaped realtime JSON: %s", out)
	}
	if !strings.Contains(out, `"endpoint":"ws://127.0.0.1:8001/connection/websocket"`) {
		t.Fatalf("dashboard missing websocket endpoint config: %s", out)
	}
}
