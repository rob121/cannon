package sites

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"

	"github.com/rob121/cannon/internal/config"
	"github.com/rob121/cannon/internal/database"
	"gorm.io/gorm"
)

type contextKey int

const siteKey contextKey = iota

// Manager resolves sites and holds runtime state.
type Manager struct {
	cfg   *config.App
	sites map[string]*config.SiteConfig
	hosts map[string]*config.SiteConfig
	mu    sync.RWMutex
}

// NewManager builds a site manager from configuration.
func NewManager(cfg *config.App) (*Manager, error) {
	m := &Manager{
		cfg:   cfg,
		sites: make(map[string]*config.SiteConfig),
		hosts: make(map[string]*config.SiteConfig),
	}

	for i := range cfg.Sites {
		site := &cfg.Sites[i]
		m.sites[site.ID] = site
		hostKey, err := hostKey(site.Host)
		if err != nil {
			return nil, fmt.Errorf("site %s host: %w", site.ID, err)
		}
		m.hosts[hostKey] = site
		if alias, ok := localHostAlias(hostKey); ok {
			m.hosts[alias] = site
		}
		if _, err := database.Connect(site); err != nil {
			return nil, fmt.Errorf("connect site %s: %w", site.ID, err)
		}
		if err := database.Migrate(site); err != nil {
			return nil, fmt.Errorf("migrate site %s: %w", site.ID, err)
		}
	}

	return m, nil
}

// Config returns the app configuration.
func (m *Manager) Config() *config.App {
	return m.cfg
}

// Reload replaces configuration and reconnects sites.
func (m *Manager) Reload(cfg *config.App) error {
	next, err := NewManager(cfg)
	if err != nil {
		return err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.cfg = next.cfg
	m.sites = next.sites
	m.hosts = next.hosts
	return nil
}

// Resolve finds the site for an HTTP request.
func (m *Manager) Resolve(r *http.Request) (*config.SiteConfig, error) {
	host := r.Host
	if override := strings.TrimSpace(r.Header.Get("X-Host")); override != "" {
		host = override
	}

	key := normalizeHost(host)
	m.mu.RLock()
	defer m.mu.RUnlock()

	if site, ok := m.hosts[key]; ok {
		return site, nil
	}

	// Treat localhost and 127.0.0.1 as equivalent for local development.
	if alias, ok := localHostAlias(key); ok {
		if site, ok := m.hosts[alias]; ok {
			return site, nil
		}
	}

	for hostKey, site := range m.hosts {
		if strings.EqualFold(hostKey, key) {
			return site, nil
		}
	}

	return nil, fmt.Errorf("no site for host %q", host)
}

// Get returns a site by id.
func (m *Manager) Get(id string) (*config.SiteConfig, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	site, ok := m.sites[id]
	return site, ok
}

// All returns configured sites.
func (m *Manager) All() []config.SiteConfig {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]config.SiteConfig, 0, len(m.sites))
	for _, site := range m.sites {
		out = append(out, *site)
	}
	return out
}

// DB returns the database for the site in context.
func DB(ctx context.Context) (*gorm.DB, error) {
	site, err := FromContext(ctx)
	if err != nil {
		return nil, err
	}
	return database.Get(site.ID)
}

// WithContext attaches a site to context.
func WithContext(ctx context.Context, site *config.SiteConfig) context.Context {
	return context.WithValue(ctx, siteKey, site)
}

// FromContext reads the site from context.
func FromContext(ctx context.Context) (*config.SiteConfig, error) {
	site, ok := ctx.Value(siteKey).(*config.SiteConfig)
	if !ok || site == nil {
		return nil, fmt.Errorf("site not found in context")
	}
	return site, nil
}

func hostKey(raw string) (string, error) {
	if !strings.Contains(raw, "://") {
		raw = "https://" + raw
	}
	u, err := url.Parse(raw)
	if err != nil {
		return "", err
	}
	return normalizeHost(u.Host), nil
}

func normalizeHost(host string) string {
	host = strings.TrimSpace(host)
	host = strings.TrimPrefix(host, "https://")
	host = strings.TrimPrefix(host, "http://")
	if idx := strings.Index(host, "/"); idx >= 0 {
		host = host[:idx]
	}
	if idx := strings.Index(host, ":"); idx >= 0 {
		host = host[:idx]
	}
	return strings.ToLower(host)
}

func localHostAlias(host string) (string, bool) {
	switch host {
	case "localhost":
		return "127.0.0.1", true
	case "127.0.0.1":
		return "localhost", true
	default:
		return "", false
	}
}
