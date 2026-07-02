package extension_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/rob121/cannon/extension"
)

func TestConfigurationEndpoint(t *testing.T) {
	store := &memoryConfigStore{data: map[string]map[string]any{}}
	schema := json.RawMessage(`{"type":"object","properties":{"name":{"type":"string"}}}`)
	ui := json.RawMessage(`{"type":"VerticalLayout","elements":[{"type":"Control","scope":"#/properties/name"}]}`)
	provider := extension.MapConfiguration([]extension.ConfigurationDefinition{
		{ID: "general", Title: "General", Schema: schema, UISchema: ui},
	}, store)

	s := extension.New(extension.Info{Name: "test", Version: "1"})
	s.OnConfiguration(provider)

	req := httptest.NewRequest(http.MethodGet, "/configuration", nil)
	rec := httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, req)

	var doc extension.ConfigurationDocument
	if err := json.NewDecoder(rec.Body).Decode(&doc); err != nil {
		t.Fatal(err)
	}
	if len(doc.Sections) != 1 || doc.Sections[0].ID != "general" {
		t.Fatalf("sections: %+v", doc.Sections)
	}

	saveBody, _ := json.Marshal(extension.ConfigurationSaveRequest{
		Section: "general",
		Data:    json.RawMessage(`{"name":"Jane"}`),
	})
	req = httptest.NewRequest(http.MethodPost, "/configuration", bytes.NewReader(saveBody))
	rec = httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("save status: %d body %s", rec.Code, rec.Body.String())
	}
	if store.data["general"]["name"] != "Jane" {
		t.Fatalf("stored: %+v", store.data)
	}

	req = httptest.NewRequest(http.MethodGet, "/capabilities", nil)
	rec = httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, req)
	var caps struct {
		Capabilities map[string]string `json:"capabilities"`
	}
	_ = json.NewDecoder(rec.Body).Decode(&caps)
	if caps.Capabilities["configuration"] != "/configuration" {
		t.Fatalf("configuration capability: %+v", caps.Capabilities)
	}
}

type memoryConfigStore struct {
	data map[string]map[string]any
}

func (m *memoryConfigStore) Load(section string) (map[string]any, error) {
	if m.data == nil {
		return map[string]any{}, nil
	}
	if data, ok := m.data[section]; ok {
		return data, nil
	}
	return map[string]any{}, nil
}

func (m *memoryConfigStore) Save(section string, data map[string]any) error {
	if m.data == nil {
		m.data = map[string]map[string]any{}
	}
	m.data[section] = data
	return nil
}
