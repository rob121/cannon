package extensions

import (
	"encoding/json"
	"io"
	"net"
	"net/http"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/rob121/cannon/internal/config"
)

type countingListener struct {
	net.Listener
	dials *atomic.Int64
}

func (c *countingListener) Accept() (net.Conn, error) {
	conn, err := c.Listener.Accept()
	if err == nil {
		c.dials.Add(1)
	}
	return conn, err
}

func startUnixHTTPServer(t *testing.T, handler http.Handler) (socketPath string, dials *atomic.Int64) {
	t.Helper()
	socketPath = filepath.Join(t.TempDir(), "ext.sock")
	ln, err := net.Listen("unix", socketPath)
	if err != nil {
		t.Fatal(err)
	}
	dials = &atomic.Int64{}
	cln := &countingListener{Listener: ln, dials: dials}
	srv := &http.Server{Handler: handler}
	go func() { _ = srv.Serve(cln) }()
	t.Cleanup(func() {
		_ = srv.Close()
		_ = ln.Close()
	})
	return socketPath, dials
}

func TestBuildWireRequestSkipsGETBody(t *testing.T) {
	m := NewManager(&config.App{}, &config.SiteConfig{ID: "site"})
	req, err := http.NewRequest(http.MethodGet, "http://example.com/x", io.NopCloser(&countingReader{n: 1024}))
	if err != nil {
		t.Fatal(err)
	}
	wire, err := m.buildWireRequest(req, nil)
	if err != nil {
		t.Fatal(err)
	}
	if wire.Body != "" {
		t.Fatalf("expected empty GET body, got %q", wire.Body)
	}
}

type countingReader struct {
	n   int
	pos int
}

func (c *countingReader) Read(p []byte) (int, error) {
	if c.pos >= c.n {
		return 0, io.EOF
	}
	n := copy(p, make([]byte, min(len(p), c.n-c.pos)))
	c.pos += n
	return n, nil
}

func TestUnixClientReusesIdleConn(t *testing.T) {
	var hits atomic.Int64
	socketPath, dials := startUnixHTTPServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		_ = json.NewEncoder(w).Encode(WireResponse{Body: "ok"})
	}))
	m := NewManager(&config.App{}, &config.SiteConfig{ID: "site"})
	for i := 0; i < 3; i++ {
		resp, err := m.do(socketPath, http.MethodPost, "/request", nil)
		if err != nil {
			t.Fatal(err)
		}
		_, _ = io.ReadAll(resp.Body)
		resp.Body.Close()
	}
	if hits.Load() != 3 {
		t.Fatalf("hits=%d", hits.Load())
	}
	if dials.Load() > 2 {
		t.Fatalf("expected connection reuse, dials=%d", dials.Load())
	}
}

func TestDoTimeoutFailsOpen(t *testing.T) {
	socketPath, _ := startUnixHTTPServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(200 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	m := NewManager(&config.App{}, &config.SiteConfig{ID: "site"})
	_, err := m.doWithTimeout(socketPath, http.MethodGet, "/slow", nil, 20*time.Millisecond)
	if err == nil {
		t.Fatal("expected timeout error")
	}
}
