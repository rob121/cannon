package extension_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/rob121/cannon/extension"
)

func TestPublicDataURLAndHandleData(t *testing.T) {
	const (
		extName = "cannon-extension-contact"
		siteID  = "example"
	)
	hash := extension.RouteHash(extName, siteID)
	wantURL := "/ext/" + hash + "/contact/submit"
	if got := extension.PublicDataURL(extName, siteID, "/contact/submit"); got != wantURL {
		t.Fatalf("PublicDataURL: got %q want %q", got, wantURL)
	}

	s := extension.New(extension.Info{Name: extName, Version: "1"})
	s.HandleData("contact/submit", func(req extension.WireRequest) extension.WireResponse {
		if extension.DataPath(req) != "contact/submit" {
			t.Fatalf("DataPath: got %q", extension.DataPath(req))
		}
		return extension.Redirect(http.StatusSeeOther, "/contact?sent=1")
	})

	body, _ := json.Marshal(extension.WireRequest{
		Method:   http.MethodPost,
		URL:      wantURL,
		SiteID:   siteID,
		DataPath: "contact/submit",
		Body:     "name=Jane",
	})
	req := httptest.NewRequest(http.MethodPost, "/data/contact/submit", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, req)

	var out extension.WireResponse
	if err := json.NewDecoder(rec.Body).Decode(&out); err != nil {
		t.Fatal(err)
	}
	if out.StatusCode != http.StatusSeeOther {
		t.Fatalf("status: got %d", out.StatusCode)
	}
}
