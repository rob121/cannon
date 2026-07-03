package accesslog

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rob121/cannon/internal/config"
)

func TestHostKey(t *testing.T) {
	if got := HostKey("http://localhost:8001"); got != "localhost-8001" {
		t.Fatalf("HostKey() = %q", got)
	}
	if got := HostKey("127.0.0.1:8001"); got != "127.0.0.1-8001" {
		t.Fatalf("HostKey() = %q", got)
	}
}

func TestWriteAndTail(t *testing.T) {
	dir := t.TempDir()
	site := &config.SiteConfig{
		ID:     "test",
		Host:   "http://localhost:8001",
		TmpDir: dir,
	}
	req := httptest.NewRequest(http.MethodGet, "/content/item/example", nil)
	req.RemoteAddr = "127.0.0.1:12345"
	req.Header.Set("User-Agent", "cannon-test")

	Write(site, req, http.StatusOK, 128, 0)
	Write(site, req, http.StatusNotFound, 0, 0)

	path := Path(site)
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("log file missing: %v", err)
	}
	content, err := Tail(path, 4096)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(content, `"GET /content/item/example"`) {
		t.Fatalf("unexpected tail content: %q", content)
	}
	if !strings.Contains(content, "404") {
		t.Fatalf("expected 404 line in tail: %q", content)
	}

	files, err := Files(site)
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 1 || files[0].Name != "access.log" {
		t.Fatalf("files: %+v", files)
	}
	if filepath.Base(files[0].Path) != "access.log" {
		t.Fatalf("path: %q", files[0].Path)
	}
}

func TestTailSkipsPartialFirstLine(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "access.log")
	longLine := strings.Repeat("x", 200) + "\nsecond line\n"
	if err := os.WriteFile(path, []byte(longLine), 0o644); err != nil {
		t.Fatal(err)
	}
	content, err := Tail(path, 20)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(content, strings.Repeat("x", 50)) {
		t.Fatalf("expected partial first line trimmed, got %q", content)
	}
	if !strings.Contains(content, "second line") {
		t.Fatalf("expected second line, got %q", content)
	}
}

func TestResolveFileRejectsUnknownName(t *testing.T) {
	dir := t.TempDir()
	site := &config.SiteConfig{ID: "test", Host: "http://example.test", TmpDir: dir}
	path := Path(site)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("line\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := ResolveFile(site, "../../etc/passwd"); err == nil {
		t.Fatal("expected error for invalid file name")
	}
}
