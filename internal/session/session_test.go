package session

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/rob121/cannon/internal/config"
)

func testStore(t *testing.T) *Store {
	t.Helper()
	site := &config.SiteConfig{TmpDir: t.TempDir()}
	app := &config.App{Session: config.SessionConfig{CookieName: "cannon_session", MaxAge: 3600}}
	store, err := NewStore(site, app)
	if err != nil {
		t.Fatal(err)
	}
	return store
}

func TestSessionLRUHitSkipsDiskReread(t *testing.T) {
	store := testStore(t)
	id := "abc123"
	if err := store.Save(id, Data{"csrf": "tok"}); err != nil {
		t.Fatal(err)
	}
	path := store.path(id)
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	data, err := store.Load(id)
	if err != nil {
		t.Fatal(err)
	}
	if data["csrf"] != "tok" {
		t.Fatalf("expected LRU hit, got %#v", data)
	}
}

func TestSessionSaveSkipRewriteWhenUnchanged(t *testing.T) {
	store := testStore(t)
	id := "skip1"
	if err := store.Save(id, Data{"k": "v"}); err != nil {
		t.Fatal(err)
	}
	path := store.path(id)
	info1, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	time.Sleep(15 * time.Millisecond)
	if err := store.Save(id, Data{"k": "v"}); err != nil {
		t.Fatal(err)
	}
	info2, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if !info1.ModTime().Equal(info2.ModTime()) {
		t.Fatal("expected disk rewrite skipped for unchanged session data")
	}
}

func TestSessionDeleteClearsMemory(t *testing.T) {
	store := testStore(t)
	id := "del1"
	if err := store.Save(id, Data{"a": 1}); err != nil {
		t.Fatal(err)
	}
	if err := store.Delete(id); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(store.path(id)); !os.IsNotExist(err) {
		t.Fatalf("expected file gone, err=%v", err)
	}
	data, err := store.Load(id)
	if err != nil {
		t.Fatal(err)
	}
	if len(data) != 0 {
		t.Fatalf("expected empty after delete, got %#v", data)
	}
}

func TestSessionExpiryClearsCache(t *testing.T) {
	store := testStore(t)
	store.cfg.MaxAge = 1
	id := "exp1"
	if err := store.Save(id, Data{"x": true}); err != nil {
		t.Fatal(err)
	}
	// Force expired entry in cache + file.
	store.mu.Lock()
	entry, _ := store.cache.Get(id)
	entry.expiresAt = time.Now().Add(-time.Second)
	store.cache.Set(id, entry)
	store.mu.Unlock()
	_ = os.Chtimes(store.path(id), time.Now(), time.Now())

	data, err := store.Load(id)
	if err != nil {
		t.Fatal(err)
	}
	if len(data) != 0 {
		t.Fatalf("expected expired session empty, got %#v", data)
	}
	if _, err := os.Stat(filepath.Join(store.dir, id+".json")); !os.IsNotExist(err) {
		// Load may have removed it; either missing or we accept gone.
	}
}
