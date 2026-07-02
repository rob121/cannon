package server

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/rob121/cannon/internal/settings"
)

func (s *Server) serveRobotsTXT(w http.ResponseWriter, r *http.Request) {
	body, err := settings.RobotsTXT(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if strings.TrimSpace(body) == "" {
		allowAI, _ := settings.AllowAICrawlers(r.Context())
		var b strings.Builder
		b.WriteString("User-agent: *\nDisallow: /admin/\n")
		if !allowAI {
			for _, ua := range []string{"GPTBot", "ChatGPT-User", "Google-Extended", "CCBot", "anthropic-ai", "ClaudeBot"} {
				b.WriteString("\nUser-agent: ")
				b.WriteString(ua)
				b.WriteString("\nDisallow: /\n")
			}
		}
		if note, _ := settings.GlobalString(r.Context(), settings.SectionSEO, "ai_crawler_policy"); strings.TrimSpace(note) != "" {
			b.WriteString("\n# ")
			b.WriteString(note)
			b.WriteString("\n")
		}
		body = b.String()
	}
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	fmt.Fprint(w, body)
}
