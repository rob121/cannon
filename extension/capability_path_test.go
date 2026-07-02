package extension_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/rob121/cannon/extension"
)

func TestPageListWithoutLeadingSlashPath(t *testing.T) {
	s := extension.New(extension.Info{Name: "test", Version: "1"})
	s.HandlePage("page", func(req extension.WireRequest) extension.WireResponse {
		return extension.HTML(200, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/page", nil)
	rec := httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /page status: got %d body %q", rec.Code, rec.Body.String())
	}

	var listed extension.PageListResponse
	if err := json.NewDecoder(rec.Body).Decode(&listed); err != nil {
		t.Fatal(err)
	}
	if len(listed.Pages) != 1 || listed.Pages[0].ID != "default" {
		t.Fatalf("pages: %+v", listed.Pages)
	}

	req = httptest.NewRequest(http.MethodGet, "/capabilities", nil)
	rec = httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, req)
	var caps struct {
		Capabilities map[string]string `json:"capabilities"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&caps); err != nil {
		t.Fatal(err)
	}
	if caps.Capabilities["page"] != "/page" {
		t.Fatalf("page capability: got %q", caps.Capabilities["page"])
	}
}
