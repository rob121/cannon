package admin

import (
	"context"
	"testing"

	"github.com/rob121/cannon/internal/config"
	"github.com/rob121/cannon/internal/security"
)

func TestBuildSiteSwitcherOptions(t *testing.T) {
	sites := []config.SiteConfig{
		{ID: "b", Name: "Beta", Host: "http://beta.test"},
		{ID: "a", Name: "Alpha", Host: "https://alpha.test"},
	}
	opts := buildSiteSwitcherOptions(sites, "b")
	if len(opts) != 2 {
		t.Fatalf("len = %d, want 2", len(opts))
	}
	if opts[0]["Label"] != "Alpha" {
		t.Fatalf("first label = %v, want Alpha (sorted)", opts[0]["Label"])
	}
	if opts[1]["Current"] != true {
		t.Fatalf("Beta should be current")
	}
	if opts[1]["AdminURL"] != "http://beta.test/admin" {
		t.Fatalf("AdminURL = %v", opts[1]["AdminURL"])
	}
}

func TestIsSuperAdmin(t *testing.T) {
	ctx := security.WithPermissions(context.Background(), security.EffectivePermissions{
		Allow: map[string]struct{}{security.PermWildcardAll: {}},
	})
	ok, err := IsSuperAdmin(ctx, 1)
	if err != nil || !ok {
		t.Fatalf("super admin: ok=%v err=%v", ok, err)
	}
	ctx = security.WithPermissions(context.Background(), security.EffectivePermissions{
		Allow: map[string]struct{}{"core.admin.sites.read": {}},
	})
	ok, err = IsSuperAdmin(ctx, 1)
	if err != nil || ok {
		t.Fatalf("sites read only should not be super admin: ok=%v err=%v", ok, err)
	}
}
