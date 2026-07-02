package extension_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"testing/fstest"

	"github.com/rob121/cannon/extension"
)

func testServer(t *testing.T) *extension.Server {
	t.Helper()
	s := extension.New(extension.Info{
		Name:          "test-extension",
		Version:       "1.0.0",
		UpdateURLBase: "https://example.com/releases",
		AdminMenuName: "Test Admin",
	})
	s.EmbedHelp(fstest.MapFS{
		"help/overview.md": {Data: []byte("# Overview\n")},
	}, "help")
	s.HandlePage("/page", func(req extension.WireRequest) extension.WireResponse {
		return extension.HTML(200, "<p>page</p>")
	})
	s.HandleAdmin("/admin", func(req extension.WireRequest) extension.WireResponse {
		return extension.HTML(200, "<p>admin</p>")
	})
	s.OnInstall(func(req extension.WireRequest) error { return nil })
	return s
}

func TestCapabilities(t *testing.T) {
	s := testServer(t)
	h := s.Handler()
	req := httptest.NewRequest(http.MethodGet, "/capabilities", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	var got struct {
		Capabilities map[string]string `json:"capabilities"`
		Defaults     struct {
			Admin struct {
				MenuName string `json:"menu_name"`
			} `json:"admin"`
		} `json:"defaults"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if got.Capabilities["page"] != "/page" {
		t.Fatalf("page capability: got %q", got.Capabilities["page"])
	}
	if got.Capabilities["admin"] != "/admin" {
		t.Fatalf("admin capability: got %q", got.Capabilities["admin"])
	}
	if got.Capabilities["help"] != "/help" {
		t.Fatalf("help capability: got %q", got.Capabilities["help"])
	}
	if _, ok := got.Capabilities["meta"]; ok {
		t.Fatalf("meta should not be listed as a capability")
	}
	if got.Defaults.Admin.MenuName != "Test Admin" {
		t.Fatalf("admin menu: got %q", got.Defaults.Admin.MenuName)
	}
}

func TestMeta(t *testing.T) {
	s := testServer(t)
	req := httptest.NewRequest(http.MethodGet, "/meta", nil)
	rec := httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, req)

	var got struct {
		Name          string `json:"name"`
		Version       string `json:"version"`
		UpdateURLBase string `json:"update_url_base"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if got.Name != "test-extension" || got.Version != "1.0.0" {
		t.Fatalf("unexpected meta: %+v", got)
	}
}

func TestInstall(t *testing.T) {
	called := false
	s := extension.New(extension.Info{Name: "x", Version: "1"})
	s.OnInstall(func(req extension.WireRequest) error {
		called = true
		if req.SiteID != "" && req.URL != "/install" {
			// site may be empty before Listen; URL comes from wire
		}
		return nil
	})

	body, _ := json.Marshal(extension.WireRequest{Method: http.MethodPost, URL: "/install", SiteID: "s1"})
	req := httptest.NewRequest(http.MethodPost, "/install", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, req)

	if !called {
		t.Fatal("install handler not called")
	}
	var out extension.WireResponse
	if err := json.NewDecoder(rec.Body).Decode(&out); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if out.StatusCode != http.StatusOK {
		t.Fatalf("status: got %d", out.StatusCode)
	}
}

func TestWirePage(t *testing.T) {
	s := testServer(t)
	body, _ := json.Marshal(extension.WireRequest{Method: http.MethodGet, URL: "/contact"})
	req := httptest.NewRequest(http.MethodPost, "/page/default", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, req)

	var out extension.WireResponse
	if err := json.NewDecoder(rec.Body).Decode(&out); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if out.Body != "<p>page</p>" {
		t.Fatalf("body: got %q", out.Body)
	}
}

func TestHelp(t *testing.T) {
	s := testServer(t)
	req := httptest.NewRequest(http.MethodGet, "/help", nil)
	rec := httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, req)

	var got struct {
		Help []string `json:"help"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(got.Help) != 1 || got.Help[0] != "/help/overview" {
		t.Fatalf("help list: %#v", got.Help)
	}

	req = httptest.NewRequest(http.MethodGet, "/help/overview", nil)
	rec = httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, req)
	if rec.Body.String() != "# Overview\n" {
		t.Fatalf("article: got %q", rec.Body.String())
	}
}

func TestParseFlags(t *testing.T) {
	flags, err := extension.ParseFlags([]string{"bin", "--site=abc", "--socket=/tmp/s.sock"})
	if err != nil {
		t.Fatal(err)
	}
	if flags.SiteID != "abc" || flags.SocketPath != "/tmp/s.sock" {
		t.Fatalf("flags: %+v", flags)
	}
}
