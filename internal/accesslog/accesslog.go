package accesslog

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/rob121/cannon/internal/config"
	"github.com/rob121/cannon/internal/sites"
)

const (
	maxFileBytes = 10 * 1024 * 1024 // 10 MiB
	maxBackups   = 5
)

type writer struct {
	path string
	mu   sync.Mutex
	file *os.File
	size int64
}

var (
	writers sync.Map // hostKey -> *writer
)

// HostKey returns a filesystem-safe key from a site host URL or host header.
func HostKey(host string) string {
	host = strings.TrimSpace(host)
	if host == "" {
		return "unknown"
	}
	if strings.Contains(host, "://") {
		if i := strings.Index(host, "://"); i >= 0 {
			host = host[i+3:]
		}
	}
	if i := strings.Index(host, "/"); i >= 0 {
		host = host[:i]
	}
	host = strings.ReplaceAll(host, ":", "-")
	host = strings.ToLower(host)
	var b strings.Builder
	for _, r := range host {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9', r == '-', r == '.':
			b.WriteRune(r)
		default:
			b.WriteRune('-')
		}
	}
	out := strings.Trim(b.String(), "-.")
	if out == "" {
		return "unknown"
	}
	return out
}

func logPath(site *config.SiteConfig) string {
	key := HostKey(site.Host)
	if key == "" {
		key = HostKey(site.ID)
	}
	dir := filepath.Join(site.TmpDir, "logs", key)
	return filepath.Join(dir, "access.log")
}

func getWriter(site *config.SiteConfig) (*writer, error) {
	key := HostKey(site.Host)
	if key == "" {
		key = site.ID
	}
	if v, ok := writers.Load(key); ok {
		return v.(*writer), nil
	}
	path := logPath(site)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, err
	}
	w := &writer{path: path}
	if err := w.open(); err != nil {
		return nil, err
	}
	actual, _ := writers.LoadOrStore(key, w)
	if actual != w {
		_ = w.file.Close()
	}
	return actual.(*writer), nil
}

func (w *writer) open() error {
	info, err := os.Stat(w.path)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	if err == nil {
		w.size = info.Size()
	}
	f, err := os.OpenFile(w.path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	w.file = f
	return nil
}

func (w *writer) rotate() error {
	if w.file != nil {
		_ = w.file.Close()
		w.file = nil
	}
	for i := maxBackups - 1; i >= 1; i-- {
		from := w.path + fmt.Sprintf(".%d", i)
		to := w.path + fmt.Sprintf(".%d", i+1)
		if _, err := os.Stat(from); err == nil {
			_ = os.Remove(to)
			_ = os.Rename(from, to)
		}
	}
	if _, err := os.Stat(w.path); err == nil {
		_ = os.Rename(w.path, w.path+".1")
	}
	w.size = 0
	return w.open()
}

func (w *writer) writeLine(line string) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.file == nil {
		if err := w.open(); err != nil {
			return err
		}
	}
	n, err := fmt.Fprintln(w.file, line)
	if err != nil {
		return err
	}
	w.size += int64(n)
	if w.size >= maxFileBytes {
		return w.rotate()
	}
	return nil
}

// Write records one HTTP access line for a site.
func Write(site *config.SiteConfig, r *http.Request, status, bytes int, duration time.Duration) {
	if site == nil || r == nil {
		return
	}
	w, err := getWriter(site)
	if err != nil {
		return
	}
	ip := r.RemoteAddr
	if fwd := r.Header.Get("X-Forwarded-For"); fwd != "" {
		ip = strings.TrimSpace(strings.Split(fwd, ",")[0])
	}
	ua := strings.ReplaceAll(r.UserAgent(), `"`, `'`)
	line := fmt.Sprintf(`%s %s "%s %s" %d %d %s "%s"`,
		time.Now().UTC().Format(time.RFC3339),
		ip,
		r.Method,
		r.URL.RequestURI(),
		status,
		bytes,
		duration.Round(time.Millisecond),
		ua,
	)
	_ = w.writeLine(line)
}

// ResponseWriter wraps http.ResponseWriter to capture status and size.
type ResponseWriter struct {
	http.ResponseWriter
	status int
	bytes  int
}

func (w *ResponseWriter) WriteHeader(code int) {
	w.status = code
	w.ResponseWriter.WriteHeader(code)
}

func (w *ResponseWriter) Write(p []byte) (int, error) {
	if w.status == 0 {
		w.status = http.StatusOK
	}
	n, err := w.ResponseWriter.Write(p)
	w.bytes += n
	return n, err
}

// Middleware logs requests for the resolved site.
func Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rw := &ResponseWriter{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(rw, r)
		site, err := sites.FromContext(r.Context())
		if err != nil {
			return
		}
		Write(site, r, rw.status, rw.bytes, time.Since(start))
	})
}
