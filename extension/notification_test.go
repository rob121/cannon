package extension

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRegisterNotificationEventCapability(t *testing.T) {
	s := New(Info{Name: "cannon-ext-test", Version: "0.0.1"})
	s.RegisterNotificationEvent(NotificationEventDefinition{
		Event:      "onThingCreated",
		Label:      "Thing created",
		Group:      "things",
		GroupLabel: "Things",
		Placeholders: []NotificationPlaceholder{
			{Key: "thing_id", Label: "ID"},
		},
	})

	req := httptest.NewRequest(http.MethodGet, "/capabilities", nil)
	rec := httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("capabilities status = %d", rec.Code)
	}
	var caps struct {
		Capabilities map[string]string `json:"capabilities"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&caps); err != nil {
		t.Fatal(err)
	}
	if caps.Capabilities["notifications"] != "/notifications" {
		t.Fatalf("notifications capability: got %#v", caps.Capabilities)
	}

	req = httptest.NewRequest(http.MethodGet, "/notifications", nil)
	rec = httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("notifications status = %d", rec.Code)
	}
	var list NotificationEventListResponse
	if err := json.NewDecoder(rec.Body).Decode(&list); err != nil {
		t.Fatal(err)
	}
	if len(list.Events) != 1 || list.Events[0].Event != "onThingCreated" {
		t.Fatalf("events = %#v", list.Events)
	}
	if len(list.Events[0].Placeholders) != 1 || list.Events[0].Placeholders[0].Key != "thing_id" {
		t.Fatalf("placeholders = %#v", list.Events[0].Placeholders)
	}
}
