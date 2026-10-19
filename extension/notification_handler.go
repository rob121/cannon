package extension

import (
	"net/http"
	"sort"
	"strings"
)

// RegisterNotificationEvent advertises a notification event Cannon may list and accept via emit_events.
func (s *Server) RegisterNotificationEvent(def NotificationEventDefinition) {
	def.Event = strings.TrimSpace(def.Event)
	if def.Event == "" {
		panic("extension: notification event name is required")
	}
	if s.notificationEvents == nil {
		s.notificationEvents = map[string]NotificationEventDefinition{}
	}
	if strings.TrimSpace(def.Label) == "" {
		def.Label = def.Event
	}
	if strings.TrimSpace(def.Group) == "" {
		def.Group = "extensions"
	}
	if strings.TrimSpace(def.GroupLabel) == "" {
		def.GroupLabel = "Extensions"
	}
	s.notificationEvents[def.Event] = def
	if s.notificationPath == "" {
		s.notificationPath = defaultNotificationPath
	}
	s.notificationPath = normalizeCapabilityPath(s.notificationPath)
}

func (s *Server) registerNotificationRoutes(mux *http.ServeMux) {
	if len(s.notificationEvents) == 0 {
		return
	}
	path := s.notificationPath
	if path == "" {
		path = defaultNotificationPath
	}
	mux.HandleFunc(path, s.handleNotificationIndex)
}

func (s *Server) handleNotificationIndex(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	writeJSON(w, http.StatusOK, s.notificationList())
}

func (s *Server) notificationList() NotificationEventListResponse {
	items := make([]NotificationEventDefinition, 0, len(s.notificationEvents))
	for _, def := range s.notificationEvents {
		items = append(items, def)
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].Group == items[j].Group {
			return items[i].Event < items[j].Event
		}
		return items[i].Group < items[j].Group
	})
	return NotificationEventListResponse{Events: items}
}
