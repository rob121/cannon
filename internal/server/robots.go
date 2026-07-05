package server

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/rob121/cannon/internal/hooks"
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
	args := map[string]any{"body": body}
	if out, err := hooks.Fire(r.Context(), hooks.OnRobotsGenerate, args); err == nil {
		body = hooks.StringArg(out, "body")
		if append := hooks.HTMLFragment(out, "robots_append"); append != "" {
			if strings.TrimSpace(body) != "" && !strings.HasSuffix(body, "\n") {
				body += "\n"
			}
			body += append
		}
	}
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	fmt.Fprint(w, body)
}
