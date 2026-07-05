package accesslog

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/rob121/cannon/internal/config"
)

const (
	maxFileBytes    = 10 * 1024 * 1024 // 10 MiB
	maxBackups      = 5
	defaultTailSize = 128 * 1024 // 128 KiB
)

type writer struct {
	path string
	mu   sync.Mutex
	file *os.File
	size int64
}

var writers sync.Map // hostKey -> *writer

// File describes one access log file on disk.
type File struct {
	Name    string
	Label   string
	Path    string
	Size    int64
	ModTime time.Time
}

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

// Path returns the active access log path for a site.
func Path(site *config.SiteConfig) string {
	return logPath(site)
}

func logDir(site *config.SiteConfig) string {
	key := HostKey(site.Host)
	if key == "" {
		key = HostKey(site.ID)
	}
	return filepath.Join(site.TmpDir, "logs", key)
}

func logPath(site *config.SiteConfig) string {
	return filepath.Join(logDir(site), "access.log")
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

// Files lists the active and rotated access log files for a site, newest first.
func Files(site *config.SiteConfig) ([]File, error) {
	if site == nil {
		return nil, fmt.Errorf("site is required")
	}
	dir := logDir(site)
	names := []string{"access.log"}
	for i := 1; i <= maxBackups; i++ {
		names = append(names, fmt.Sprintf("access.log.%d", i))
	}
	out := make([]File, 0, len(names))
	for _, name := range names {
		path := filepath.Join(dir, name)
		info, err := os.Stat(path)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, err
		}
		if info.IsDir() {
			continue
		}
		out = append(out, File{
			Name:    name,
			Label:   fileLabel(name),
			Path:    path,
			Size:    info.Size(),
			ModTime: info.ModTime(),
		})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Name == "access.log" {
			return true
		}
		if out[j].Name == "access.log" {
			return false
		}
		return out[i].Name < out[j].Name
	})
	return out, nil
}

func fileLabel(name string) string {
	if name == "access.log" {
		return "Current"
	}
	return name
}

// ResolveFile returns a log file path from a site and requested file name.
func ResolveFile(site *config.SiteConfig, name string) (File, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		name = "access.log"
	}
	files, err := Files(site)
	if err != nil {
		return File{}, err
	}
	for _, file := range files {
		if file.Name == name {
			return file, nil
		}
	}
	return File{}, os.ErrNotExist
}

// Tail reads up to maxBytes from the end of a log file.
func Tail(path string, maxBytes int64) (string, error) {
	if maxBytes <= 0 {
		maxBytes = defaultTailSize
	}
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	info, err := f.Stat()
	if err != nil {
		return "", err
	}
	if info.Size() == 0 {
		return "", nil
	}
	start := int64(0)
	if info.Size() > maxBytes {
		start = info.Size() - maxBytes
	}
	if _, err := f.Seek(start, io.SeekStart); err != nil {
		return "", err
	}
	data, err := io.ReadAll(f)
	if err != nil {
		return "", err
	}
	if start > 0 {
		if idx := strings.IndexByte(string(data), '\n'); idx >= 0 {
			data = data[idx+1:]
		}
	}
	return string(data), nil
}

// ResponseWriter wraps http.ResponseWriter to capture status and size.
type ResponseWriter struct {
	http.ResponseWriter
	Status   int
	Bytes    int
	Hijacked bool
}

// Hijack delegates to the underlying ResponseWriter when supported.
func (w *ResponseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	h, ok := w.ResponseWriter.(http.Hijacker)
	if !ok {
		return nil, nil, errors.New("accesslog: underlying ResponseWriter does not implement http.Hijacker")
	}
	w.Hijacked = true
	return h.Hijack()
}

// Flush delegates to the underlying ResponseWriter when supported.
func (w *ResponseWriter) Flush() {
	if f, ok := w.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

func (w *ResponseWriter) WriteHeader(code int) {
	w.Status = code
	w.ResponseWriter.WriteHeader(code)
}

func (w *ResponseWriter) Write(p []byte) (int, error) {
	if w.Status == 0 {
		w.Status = http.StatusOK
	}
	n, err := w.ResponseWriter.Write(p)
	w.Bytes += n
	return n, err
}
