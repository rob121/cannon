package session

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/rob121/cannon/internal/config"
	"github.com/rob121/cannon/internal/lrucache"
)

// MaxCachedSessions is the per-site in-memory session LRU capacity.
var MaxCachedSessions = 2048

type cachedSession struct {
	expiresAt time.Time
	data      Data
	raw       []byte // last marshaled payload written/read (for skip-rewrite)
}

// Store manages file-based sessions per site with an in-memory LRU fronting disk.
type Store struct {
	cfg   *config.SessionConfig
	dir   string
	mu    sync.Mutex
	cache *lrucache.LRU[string, cachedSession]
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
	return &Store{
		cfg:   &cfg,
		dir:   dir,
		cache: lrucache.New[string, cachedSession](MaxCachedSessions),
	}, nil
}

// Create returns a new session id.
func (s *Store) Create() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}

// Load reads session data from memory or disk.
func (s *Store) Load(id string) (Data, error) {
	if id == "" {
		return Data{}, nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	if entry, ok := s.cache.Get(id); ok {
		if now.After(entry.expiresAt) {
			s.cache.Delete(id)
			_ = os.Remove(s.path(id))
			return Data{}, nil
		}
		return cloneData(entry.data), nil
	}

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
	if now.After(payload.ExpiresAt) {
		_ = os.Remove(path)
		return Data{}, nil
	}
	cloned := cloneData(payload.Data)
	s.cache.Set(id, cachedSession{
		expiresAt: payload.ExpiresAt,
		data:      cloned,
		raw:       append([]byte(nil), raw...),
	})
	return cloneData(cloned), nil
}

// Save persists session data (write-through LRU + disk).
func (s *Store) Save(id string, data Data) error {
	if id == "" {
		return fmt.Errorf("session id required")
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	cloned := cloneData(data)
	expiresAt := time.Now().Add(time.Duration(s.cfg.MaxAge) * time.Second)
	payload := struct {
		ExpiresAt time.Time `json:"expires_at"`
		Data      Data      `json:"data"`
	}{
		ExpiresAt: expiresAt,
		Data:      cloned,
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	// Skip disk rewrite when data payload is unchanged and expiry still has > half TTL remaining.
	if prev, ok := s.cache.Peek(id); ok && len(prev.raw) > 0 {
		if dataEqualIgnoringExpiry(prev.raw, raw) && time.Until(prev.expiresAt) > time.Duration(s.cfg.MaxAge)*time.Second/2 {
			s.cache.Set(id, cachedSession{
				expiresAt: prev.expiresAt,
				data:      cloned,
				raw:       prev.raw,
			})
			return nil
		}
	}

	if err := os.WriteFile(s.path(id), raw, 0600); err != nil {
		return err
	}
	s.cache.Set(id, cachedSession{
		expiresAt: expiresAt,
		data:      cloned,
		raw:       raw,
	})
	return nil
}

// Delete removes a session from memory and disk.
func (s *Store) Delete(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.cache.Delete(id)
	err := os.Remove(s.path(id))
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
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

func cloneData(data Data) Data {
	if data == nil {
		return Data{}
	}
	out := make(Data, len(data))
	for k, v := range data {
		out[k] = v
	}
	return out
}

func dataEqualIgnoringExpiry(prev, next []byte) bool {
	var a, b struct {
		Data Data `json:"data"`
	}
	if json.Unmarshal(prev, &a) != nil || json.Unmarshal(next, &b) != nil {
		return false
	}
	ra, err1 := json.Marshal(a.Data)
	rb, err2 := json.Marshal(b.Data)
	if err1 != nil || err2 != nil {
		return false
	}
	return bytes.Equal(ra, rb)
}
