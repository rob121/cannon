package session

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/rob121/cannon/internal/config"
)

// Store manages file-based sessions per site.
type Store struct {
	cfg *config.SessionConfig
	dir string
	mu  sync.Mutex
}

// Data holds session values.
type Data map[string]any

// NewStore creates a session store for a site tmp directory.
func NewStore(site *config.SiteConfig, app *config.App) (*Store, error) {
	dir := filepath.Join(site.TmpDir, "sessions")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("create session dir: %w", err)
	}
	cfg := app.Session
	return &Store{cfg: &cfg, dir: dir}, nil
}

// Create returns a new session id.
func (s *Store) Create() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}

// Load reads session data from disk.
func (s *Store) Load(id string) (Data, error) {
	if id == "" {
		return Data{}, nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	path := s.path(id)
	raw, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return Data{}, nil
		}
		return nil, err
	}

	var payload struct {
		ExpiresAt time.Time `json:"expires_at"`
		Data      Data      `json:"data"`
	}
	if err := json.Unmarshal(raw, &payload); err != nil {
		return nil, err
	}
	if time.Now().After(payload.ExpiresAt) {
		_ = os.Remove(path)
		return Data{}, nil
	}
	return payload.Data, nil
}

// Save persists session data.
func (s *Store) Save(id string, data Data) error {
	if id == "" {
		return fmt.Errorf("session id required")
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	payload := struct {
		ExpiresAt time.Time `json:"expires_at"`
		Data      Data      `json:"data"`
	}{
		ExpiresAt: time.Now().Add(time.Duration(s.cfg.MaxAge) * time.Second),
		Data:      data,
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	return os.WriteFile(s.path(id), raw, 0600)
}

// Delete removes a session file.
func (s *Store) Delete(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return os.Remove(s.path(id))
}

// CookieName returns the configured cookie name.
func (s *Store) CookieName() string {
	return s.cfg.CookieName
}

// MaxAge returns cookie max age in seconds.
func (s *Store) MaxAge() int {
	return s.cfg.MaxAge
}

func (s *Store) path(id string) string {
	return filepath.Join(s.dir, id+".json")
}
