package extension_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/rob121/cannon/extension"
)

func TestPageListAndRender(t *testing.T) {
	s := extension.New(extension.Info{Name: "test", Version: "1"})
	s.RegisterPage(extension.PageDefinition{
		ID:    "contact",
		Title: "Contact Page",
		Fields: []extension.PageField{
			{Name: "form_id", Label: "Form", Type: "number"},
		},
	}, func(item string, req extension.WireRequest) extension.WireResponse {
		if item != "contact" {
			t.Fatalf("item: got %q", item)
		}
		if extension.PageItem(req) != "contact" {
			t.Fatalf("page item: got %q", extension.PageItem(req))
		}
		formID, ok := extension.PageDataInt(req, "form_id")
		if !ok || formID != 42 {
			t.Fatalf("form_id: got %d ok=%v", formID, ok)
		}
		return extension.HTML(200, "<main>contact</main>")
	})

	req := httptest.NewRequest(http.MethodGet, "/page", nil)
	rec := httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, req)

	var listed extension.PageListResponse
	if err := json.NewDecoder(rec.Body).Decode(&listed); err != nil {
		t.Fatal(err)
	}
	if len(listed.Pages) != 1 || listed.Pages[0].ID != "contact" {
		t.Fatalf("pages: %+v", listed.Pages)
	}

	body, _ := json.Marshal(extension.WireRequest{
		Method:   http.MethodGet,
		URL:      "/contact",
		SiteID:   "site-1",
		PageItem: "contact",
		PageData: map[string]any{"form_id": 42},
	})
	req = httptest.NewRequest(http.MethodPost, "/page/contact", bytes.NewReader(body))
	rec = httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, req)

	var out extension.WireResponse
	if err := json.NewDecoder(rec.Body).Decode(&out); err != nil {
		t.Fatal(err)
	}
	if out.Body != "<main>contact</main>" {
		t.Fatalf("body: got %q", out.Body)
	}
}

func TestHandlePageDefault(t *testing.T) {
	s := extension.New(extension.Info{Name: "test", Version: "1"})
	s.HandlePage("/page", func(req extension.WireRequest) extension.WireResponse {
		return extension.HTML(200, "default page")
	})

	req := httptest.NewRequest(http.MethodGet, "/capabilities", nil)
	rec := httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, req)
	var caps struct {
		Capabilities map[string]string `json:"capabilities"`
	}
	_ = json.NewDecoder(rec.Body).Decode(&caps)
	if caps.Capabilities["page"] != "/page" {
		t.Fatalf("page cap: %+v", caps.Capabilities)
	}

	body, _ := json.Marshal(extension.WireRequest{Method: http.MethodGet, URL: "/home"})
	req = httptest.NewRequest(http.MethodPost, "/page/default", bytes.NewReader(body))
	rec = httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, req)
	var out extension.WireResponse
	if err := json.NewDecoder(rec.Body).Decode(&out); err != nil {
		t.Fatal(err)
	}
	if out.Body != "default page" {
		t.Fatalf("body: got %q", out.Body)
	}
}
