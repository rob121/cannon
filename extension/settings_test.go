package extension_test

import (
	"testing"
	"time"

	"github.com/rob121/cannon/extension"
)

func TestDevelopmentMode(t *testing.T) {
	if extension.DevelopmentMode(extension.WireRequest{}) {
		t.Fatal("expected false when unset")
	}
	if !extension.DevelopmentMode(extension.WireRequest{Settings: map[string]string{extension.SettingDevelopmentMode: "true"}}) {
		t.Fatal("expected true")
	}
	if extension.DevelopmentMode(extension.WireRequest{Settings: map[string]string{extension.SettingDevelopmentMode: "false"}}) {
		t.Fatal("expected false")
	}
}

func TestFormatDateUsesWireSettings(t *testing.T) {
	req := extension.WireRequest{
		Settings: map[string]string{
			extension.SettingDateFormat: "2006-01-02",
		},
	}
	ts := time.Date(2026, 10, 17, 15, 4, 5, 0, time.UTC)
	if got := extension.FormatDate(req, ts); got != "2026-10-17" {
		t.Fatalf("FormatDate = %q", got)
	}
	if got := extension.FormatDate(extension.WireRequest{}, ts); got != "10/17/2026" {
		t.Fatalf("default FormatDate = %q", got)
	}
	if got := extension.FormatDatePtr(req, nil); got != "" {
		t.Fatalf("nil ptr = %q", got)
	}
}
