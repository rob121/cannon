package api

import (
	"context"
	"net/http"
	"strings"
	"sync"
	"time"
)

type bucket struct {
	count   int
	resetAt time.Time
}

var rateMu sync.Mutex
var rateBuckets = map[string]*bucket{}

// AllowRate reports whether a request is within the configured limit.
func AllowRate(ctx context.Context, key string, limit int) (bool, int) {
	if limit <= 0 {
		limit = 120
	}
	now := time.Now()
	rateMu.Lock()
	defer rateMu.Unlock()
	b, ok := rateBuckets[key]
	if !ok || now.After(b.resetAt) {
		rateBuckets[key] = &bucket{count: 1, resetAt: now.Add(time.Minute)}
		return true, limit - 1
	}
	if b.count >= limit {
		retry := int(time.Until(b.resetAt).Seconds())
		if retry < 1 {
			retry = 1
		}
		return false, retry
	}
	b.count++
	return true, limit - b.count
}

func clientIP(r *http.Request) string {
	if r == nil {
		return ""
	}
	if xff := strings.TrimSpace(r.Header.Get("X-Forwarded-For")); xff != "" {
		parts := strings.Split(xff, ",")
		return strings.TrimSpace(parts[0])
	}
	host := r.RemoteAddr
	if i := strings.LastIndex(host, ":"); i > 0 {
		return host[:i]
	}
	return host
}
