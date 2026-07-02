package extension_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/rob121/cannon/extension"
)

func TestEndpointListAndHandle(t *testing.T) {
	s := extension.New(extension.Info{Name: "test", Version: "1"})
	s.RegisterEndpoint(extension.EndpointDefinition{
		ID:    "submit",
		Title: "Submit Contact Form",
	}, func(item string, req extension.WireRequest) extension.WireResponse {
		if item != "submit" {
			t.Fatalf("item: got %q", item)
		}
		if req.Method != http.MethodPost || req.URL != "/contact/submit" {
			t.Fatalf("request: %+v", req)
		}
		return extension.Redirect(http.StatusSeeOther, "/contact?sent=1")
	})

	req := httptest.NewRequest(http.MethodGet, "/endpoint", nil)
	rec := httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, req)

	var listed extension.EndpointListResponse
	if err := json.NewDecoder(rec.Body).Decode(&listed); err != nil {
		t.Fatal(err)
	}
	if len(listed.Endpoints) != 1 || listed.Endpoints[0].ID != "submit" {
		t.Fatalf("endpoints: %+v", listed.Endpoints)
	}

	body, _ := json.Marshal(extension.WireRequest{
		Method:       http.MethodPost,
		URL:          "/contact/submit",
		Body:         "name=Jane",
		SiteID:       "site-1",
		EndpointItem: "submit",
	})
	req = httptest.NewRequest(http.MethodPost, "/endpoint/submit", bytes.NewReader(body))
	rec = httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, req)

	var out extension.WireResponse
	if err := json.NewDecoder(rec.Body).Decode(&out); err != nil {
		t.Fatal(err)
	}
	if out.StatusCode != http.StatusSeeOther {
		t.Fatalf("status: got %d", out.StatusCode)
	}
	if loc := out.Header["Location"]; len(loc) != 1 || loc[0] != "/contact?sent=1" {
		t.Fatalf("location: %+v", out.Header)
	}
}
