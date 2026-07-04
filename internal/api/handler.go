package api

import (
	"net/http"
	"strings"

	"github.com/rob121/cannon/internal/middleware"
)

// Handler serves /api/v1/* JSON endpoints.
type Handler struct {
	chain *middleware.Chain
}

// NewHandler creates the Content API handler.
func NewHandler(chain *middleware.Chain) *Handler {
	return &Handler{chain: chain}
}

// ServeHTTP implements http.Handler.
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	origins, _ := CORSOrigins(ctx)
	if applyCORS(w, r, origins) {
		return
	}

	path := strings.TrimPrefix(r.URL.Path, "/api/v1")
	path = strings.Trim(path, "/")
	parts := strings.Split(path, "/")

	// Public within version: docs + openapi
	if len(parts) > 0 && parts[0] == "docs" {
		serveDocs(w, r)
		return
	}
	if path == "openapi.json" {
		serveOpenAPI(w, r)
		return
	}

	key := extractAPIKey(r)
	if key == "" {
		writeError(w, http.StatusUnauthorized, "unauthorized", "Missing or invalid API key")
		return
	}
	cred, err := ValidateAPIKey(ctx, key)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "unauthorized", "Invalid or missing API key")
		return
	}
	ctx = WithCredentialID(ctx, cred.CredentialID)

	limit, _ := defaultRateLimit(ctx)
	rateKey := "cred:" + strings.TrimSpace(cred.TokenPrefix) + ":" + clientIP(r)
	if ok, retry := AllowRate(ctx, rateKey, limit); !ok {
		writeRateLimited(w, retry)
		return
	}

	if jwt := extractBearerJWT(r); jwt != "" {
		userID, err := ParseAccessToken(ctx, jwt)
		if err != nil {
			writeError(w, http.StatusUnauthorized, "unauthorized", "Invalid or expired access token")
			return
		}
		ctx = WithUserID(ctx, userID)
	}
	viewerGroups, err := resolveViewerGroups(ctx)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "server_error", err.Error())
		return
	}
	ctx = WithViewerGroups(ctx, viewerGroups)
	r = r.WithContext(ctx)

	if len(parts) == 0 {
		writeJSON(w, http.StatusOK, map[string]any{"name": "Cannon Content API", "version": "v1"})
		return
	}

	switch parts[0] {
	case "auth":
		h.serveAuth(w, r, parts[1:])
	case "items":
		h.serveItems(w, r, parts[1:])
	case "categories":
		h.serveCategories(w, r, parts[1:])
	case "tags":
		h.serveTags(w, r, parts[1:])
	case "search":
		h.serveSearch(w, r)
	case "media":
		h.serveMedia(w, r, parts[1:])
	case "authors":
		h.serveAuthors(w, r, parts[1:])
	default:
		writeError(w, http.StatusNotFound, "not_found", "Unknown endpoint")
	}
}

func (h *Handler) requireJWT(w http.ResponseWriter, r *http.Request) (uint, bool) {
	userID, ok := UserID(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized", "Sign in required")
		return 0, false
	}
	return userID, true
}
