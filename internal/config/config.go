package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

const ConfigName = "sites"

const configEnv = "CANNON_CONFIG"

// UpdateConfig holds Cannon application update settings.
type UpdateConfig struct {
	URLBase string `json:"url_base"`
}

// App holds the parsed sites.json configuration.
type App struct {
	InstallEnabled bool             `json:"install_enabled"`
	DataRoot       string           `json:"data_root"`
	Update         UpdateConfig     `json:"update"`
	Extensions     ExtensionsConfig `json:"extensions"`
	Session        SessionConfig    `json:"session"`
	Sites          []SiteConfig     `json:"sites"`
}

// ExtensionsConfig holds global extension paths.
type ExtensionsConfig struct {
	Dir        string `json:"dir"`
	SocketsDir string `json:"sockets_dir"`
}

// SessionConfig holds session defaults.
type SessionConfig struct {
	CookieName string `json:"cookie_name"`
	MaxAge     int    `json:"max_age"`
}

// SiteConfig holds per-site settings.
type SiteConfig struct {
	ID          string         `json:"id"`
	Name        string         `json:"name"`
	Host        string         `json:"host"`
	Database    DatabaseConfig `json:"database"`
	TemplateDir string         `json:"template_dir"`
	AssetsDir   string         `json:"assets_dir"`
	TmpDir      string         `json:"tmp_dir"`
	LanguageDir string         `json:"language_dir"`
}

// DatabaseConfig holds database connection settings.
type DatabaseConfig struct {
	Type string `json:"type"`
	DSN  string `json:"dsn"`
}

var (
	mu         sync.RWMutex
	appCfg     *App
	loadedPath string
)

// Load reads sites.json from the first matching search path.
func Load() (*App, string, error) {
	cfg := defaultApp()
	loadedFrom := ""

	for _, path := range configSearchPaths() {
		if _, err := os.Stat(path); err != nil {
			continue
		}
		raw, err := os.ReadFile(path)
		if err != nil {
			return nil, "", fmt.Errorf("read config %s: %w", path, err)
		}
		if err := json.Unmarshal(raw, cfg); err != nil {
			return nil, "", fmt.Errorf("parse config %s: %w", path, err)
		}
		loadedFrom = path
		break
	}

	applyDefaults(cfg)

	mu.Lock()
	appCfg = cfg
	loadedPath = loadedFrom
	mu.Unlock()
	return cfg, loadedFrom, nil
}

// ConfigPath returns the path to the loaded sites.json, if any.
func ConfigPath() string {
	mu.RLock()
	defer mu.RUnlock()
	return loadedPath
}

// LoadFile reads sites.json from an explicit path.
func LoadFile(path string) (*App, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}
	raw, err := os.ReadFile(abs)
	if err != nil {
		return nil, fmt.Errorf("read config %s: %w", abs, err)
	}
	cfg := defaultApp()
	if err := json.Unmarshal(raw, cfg); err != nil {
		return nil, fmt.Errorf("parse config %s: %w", abs, err)
	}
	applyDefaults(cfg)
	return cfg, nil
}

// FindSite returns the site config for an id.
func FindSite(app *App, id string) (*SiteConfig, error) {
	if app == nil {
		return nil, fmt.Errorf("config is required")
	}
	id = strings.TrimSpace(id)
	for i := range app.Sites {
		if app.Sites[i].ID == id {
			return &app.Sites[i], nil
		}
	}
	return nil, fmt.Errorf("site %q not found", id)
}

// Reload reads the configuration from disk again.
func Reload() (*App, string, error) {
	return Load()
}

func defaultApp() *App {
	return &App{
		InstallEnabled: true,
		Session: SessionConfig{
			CookieName: "cannon_session",
			MaxAge:     86400,
		},
		Extensions: ExtensionsConfig{
			Dir:        "./extensions",
			SocketsDir: "./sockets",
		},
	}
}

func searchPaths() []string {
	paths := []string{filepath.Join(".", ConfigName+".json")}
	if exe, err := os.Executable(); err == nil {
		paths = append(paths, filepath.Join(filepath.Dir(exe), ConfigName+".json"))
	}
	if home, err := os.UserHomeDir(); err == nil {
		paths = append(paths, filepath.Join(home, ".cannon", ConfigName+".json"))
	}
	paths = append(paths, filepath.Join("/etc/cannon", ConfigName+".json"))
	return paths
}

func configSearchPaths() []string {
	candidates := []string{}
	if env := strings.TrimSpace(os.Getenv(configEnv)); env != "" {
		candidates = append(candidates, env)
	}
	candidates = append(candidates, searchPaths()...)

	seen := make(map[string]struct{}, len(candidates))
	out := make([]string, 0, len(candidates))
	for _, path := range candidates {
		abs, err := filepath.Abs(path)
		if err != nil {
			continue
		}
		if _, ok := seen[abs]; ok {
			continue
		}
		seen[abs] = struct{}{}
		out = append(out, abs)
	}
	return out
}

func resolveSavePath() (string, error) {
	if path := ConfigPath(); path != "" {
		return path, nil
	}
	if env := strings.TrimSpace(os.Getenv(configEnv)); env != "" {
		return filepath.Abs(env)
	}
	for _, path := range configSearchPaths() {
		if _, err := os.Stat(path); err == nil {
			return path, nil
		}
	}
	return filepath.Abs(filepath.Join(".", ConfigName+".json"))
}

// Get returns the loaded configuration.
func Get() *App {
	mu.RLock()
	defer mu.RUnlock()
	return appCfg
}

// Save writes the configuration back to the loaded path, or the first existing
// config file discovered in the search path order.
func Save(cfg *App) error {
	applyDefaults(cfg)
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}
	path, err := resolveSavePath()
	if err != nil {
		return err
	}
	if dir := filepath.Dir(path); dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("create config dir: %w", err)
		}
	}
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("write config %s: %w", path, err)
	}
	mu.Lock()
	appCfg = cfg
	loadedPath = path
	mu.Unlock()
	return nil
}

// NeedsInstall returns true when install wizard should run.
func NeedsInstall(cfg *App) bool {
	if cfg == nil {
		return true
	}
	if len(cfg.Sites) > 0 && !cfg.InstallEnabled {
		return false
	}
	return cfg.InstallEnabled || len(cfg.Sites) == 0
}

func applyDefaults(cfg *App) {
	if cfg.DataRoot == "" {
		cfg.DataRoot = "./data"
	}
	if cfg.Session.CookieName == "" {
		cfg.Session.CookieName = "cannon_session"
	}
	if cfg.Session.MaxAge == 0 {
		cfg.Session.MaxAge = 86400
	}
	if cfg.Extensions.Dir == "" {
		cfg.Extensions.Dir = filepath.Join(cfg.DataRoot, "extensions")
	}
	if cfg.Extensions.SocketsDir == "" {
		cfg.Extensions.SocketsDir = filepath.Join(cfg.DataRoot, "sockets")
	}
	if strings.TrimSpace(cfg.Update.URLBase) == "" {
		cfg.Update.URLBase = "https://github.com/rob121/cannon/releases/download"
	}
}
