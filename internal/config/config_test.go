package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestSaveUsesLoadedConfigPath(t *testing.T) {
	root := t.TempDir()
	etcDir := filepath.Join(root, "etc", "cannon")
	if err := os.MkdirAll(etcDir, 0755); err != nil {
		t.Fatal(err)
	}
	loaded := filepath.Join(etcDir, "sites.json")
	if err := os.WriteFile(loaded, []byte(`{"install_enabled":true,"sites":[]}`), 0644); err != nil {
		t.Fatal(err)
	}

	cwd := filepath.Join(root, "work")
	if err := os.MkdirAll(cwd, 0755); err != nil {
		t.Fatal(err)
	}
	t.Chdir(cwd)

	cfg, err := LoadFile(loaded)
	if err != nil {
		t.Fatal(err)
	}
	mu.Lock()
	appCfg = cfg
	loadedPath = loaded
	mu.Unlock()

	cfg.DataRoot = "/var/lib/cannon/data"
	cfg.Sites = []SiteConfig{{
		ID:   "example",
		Name: "Example",
		Host: "https://example.test",
		Database: DatabaseConfig{
			Type: "sqlite",
			DSN:  "/var/lib/cannon/data/example/example.sqlite",
		},
	}}
	if err := Save(cfg); err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat(filepath.Join(cwd, "sites.json")); err == nil {
		t.Fatal("expected install save not to create sites.json in cwd")
	}
	raw, err := os.ReadFile(loaded)
	if err != nil {
		t.Fatal(err)
	}
	var saved App
	if err := json.Unmarshal(raw, &saved); err != nil {
		t.Fatal(err)
	}
	if len(saved.Sites) != 1 || saved.Sites[0].ID != "example" {
		t.Fatalf("saved config = %+v", saved)
	}
	if ConfigPath() != loaded {
		t.Fatalf("ConfigPath = %q, want %q", ConfigPath(), loaded)
	}
}

func TestResolveSavePathPrefersExistingSearchPath(t *testing.T) {
	root := t.TempDir()
	cwd := filepath.Join(root, "work")
	if err := os.MkdirAll(cwd, 0755); err != nil {
		t.Fatal(err)
	}
	t.Chdir(cwd)

	existing := filepath.Join(cwd, "sites.json")
	if err := os.WriteFile(existing, []byte(`{"install_enabled":true}`), 0644); err != nil {
		t.Fatal(err)
	}

	mu.Lock()
	appCfg = defaultApp()
	loadedPath = ""
	mu.Unlock()

	got, err := resolveSavePath()
	if err != nil {
		t.Fatal(err)
	}
	if got != existing {
		t.Fatalf("resolveSavePath = %q, want %q", got, existing)
	}
}

func TestLoadHonorsCannonConfigEnv(t *testing.T) {
	root := t.TempDir()
	cfgPath := filepath.Join(root, "custom", "sites.json")
	if err := os.MkdirAll(filepath.Dir(cfgPath), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(cfgPath, []byte(`{"install_enabled":false,"sites":[{"id":"env","name":"Env","host":"http://env.test","database":{"type":"sqlite","dsn":"`+filepath.Join(root, "env.sqlite")+`"}}]}`), 0644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("CANNON_CONFIG", cfgPath)

	cfg, loaded, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if loaded != cfgPath {
		t.Fatalf("loaded = %q, want %q", loaded, cfgPath)
	}
	if len(cfg.Sites) != 1 || cfg.Sites[0].ID != "env" {
		t.Fatalf("cfg = %+v", cfg)
	}
}
