package extension_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/rob121/cannon/extension"
)

func TestBlockListAndRender(t *testing.T) {
	s := extension.New(extension.Info{Name: "test", Version: "1"})
	s.RegisterBlock(extension.BlockDefinition{
		ID:     "contact-form",
		Title:  "Contact Form",
		Spaces: []string{"footer"},
	}, func(item string, req extension.WireRequest) extension.WireResponse {
		if item != "contact-form" {
			t.Fatalf("item: got %q", item)
		}
		if req.BlockSpace != "footer" {
			t.Fatalf("space: got %q", req.BlockSpace)
		}
		return extension.HTML(200, "<footer>contact</footer>")
	})

	req := httptest.NewRequest(http.MethodGet, "/block", nil)
	rec := httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, req)

	var listed extension.BlockListResponse
	if err := json.NewDecoder(rec.Body).Decode(&listed); err != nil {
		t.Fatal(err)
	}
	if len(listed.Blocks) != 1 || listed.Blocks[0].ID != "contact-form" {
		t.Fatalf("blocks: %+v", listed.Blocks)
	}

	body, _ := json.Marshal(extension.WireRequest{
		Method:     http.MethodGet,
		URL:        "/",
		SiteID:     "site-1",
		BlockSpace: "footer",
		BlockItem:  "contact-form",
	})
	req = httptest.NewRequest(http.MethodPost, "/block/contact-form", bytes.NewReader(body))
	rec = httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, req)

	var out extension.WireResponse
	if err := json.NewDecoder(rec.Body).Decode(&out); err != nil {
		t.Fatal(err)
	}
	if out.Body != "<footer>contact</footer>" {
		t.Fatalf("body: got %q", out.Body)
	}
}

func TestHandleBlockDefault(t *testing.T) {
	s := extension.New(extension.Info{Name: "test", Version: "1"})
	s.HandleBlock("/block", func(req extension.WireRequest) extension.WireResponse {
		return extension.HTML(200, "default block")
	})

	req := httptest.NewRequest(http.MethodGet, "/capabilities", nil)
	rec := httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, req)
	var caps struct {
		Capabilities map[string]string `json:"capabilities"`
	}
	_ = json.NewDecoder(rec.Body).Decode(&caps)
	if caps.Capabilities["block"] != "/block" {
		t.Fatalf("block cap: %+v", caps.Capabilities)
	}
}
