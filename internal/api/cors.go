package api

import (
	"context"
	"net/http"
	"strconv"
	"strings"

	"github.com/rob121/cannon/internal/groups"
	"github.com/rob121/cannon/internal/settings"
)

func applyCORS(w http.ResponseWriter, r *http.Request, origins []string) bool {
	origin := strings.TrimSpace(r.Header.Get("Origin"))
	if origin == "" || len(origins) == 0 {
		return false
	}
	allowed := false
	for _, o := range origins {
		if o == "*" || strings.EqualFold(o, origin) {
			allowed = true
			break
		}
	}
	if !allowed {
		return false
	}
	w.Header().Set("Access-Control-Allow-Origin", origin)
	w.Header().Set("Vary", "Origin")
	w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type, X-Cannon-API-Key")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)
		return true
	}
	return false
}

func resolveViewerGroups(ctx context.Context) ([]uint, error) {
	if cached, ok := ViewerGroups(ctx); ok {
		return cached, nil
	}
	if userID, ok := UserID(ctx); ok {
		return groups.UserGroupIDs(ctx, userID)
	}
	dbGroups, err := groups.ViewerGroupIDs(ctx)
	if err != nil {
		return nil, err
	}
	return dbGroups, nil
}

func defaultRateLimit(ctx context.Context) (int, error) {
	return settings.GlobalIntDefault(ctx, sectionAPI, "rate_limit_per_minute", 120)
}

func loginRateLimit(ctx context.Context) (int, error) {
	return settings.GlobalIntDefault(ctx, sectionAPI, "login_rate_limit_per_minute", 20)
}

func writeRateLimited(w http.ResponseWriter, retry int) {
	w.Header().Set("Retry-After", strconv.Itoa(retry))
	writeError(w, http.StatusTooManyRequests, "rate_limited", "Too many requests. Try again later.")
}
