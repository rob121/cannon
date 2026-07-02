package content

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/rob121/cannon/internal/models"
	"github.com/rob121/cannon/internal/sites"
)

var (
	ErrCommentSpam = errors.New("comment rejected as spam")

	commentRateMu sync.Mutex
	commentRates  = map[string][]time.Time{}
)

const (
	commentHoneypotField = "website"
	commentRateLimit     = 5
	commentRateWindow    = 10 * time.Minute
)

// ValidateCommentSpam rejects honeypot submissions and simple IP rate limits.
func ValidateCommentSpam(r *http.Request, ip string) error {
	if strings.TrimSpace(r.FormValue(commentHoneypotField)) != "" {
		return ErrCommentSpam
	}
	ip = strings.TrimSpace(ip)
	if ip == "" {
		return nil
	}
	now := time.Now()
	cutoff := now.Add(-commentRateWindow)
	commentRateMu.Lock()
	defer commentRateMu.Unlock()
	times := commentRates[ip]
	filtered := times[:0]
	for _, t := range times {
		if t.After(cutoff) {
			filtered = append(filtered, t)
		}
	}
	if len(filtered) >= commentRateLimit {
		return fmt.Errorf("%w: too many comments from this address", ErrCommentSpam)
	}
	filtered = append(filtered, now)
	commentRates[ip] = filtered
	return nil
}

// PruneCommentRateLimits drops stale IP entries (for tests).
func PruneCommentRateLimits() {
	cutoff := time.Now().Add(-commentRateWindow)
	commentRateMu.Lock()
	defer commentRateMu.Unlock()
	for ip, times := range commentRates {
		filtered := times[:0]
		for _, t := range times {
			if t.After(cutoff) {
				filtered = append(filtered, t)
			}
		}
		if len(filtered) == 0 {
			delete(commentRates, ip)
			continue
		}
		commentRates[ip] = filtered
	}
}

// ListAuthorsWithItems returns active users who have published visible items.
func ListAuthorsWithItems(ctx context.Context, viewerGroups []uint) ([]models.User, error) {
	q, err := VisibleItemsQuery(ctx, viewerGroups)
	if err != nil {
		return nil, err
	}
	var authorIDs []uint
	if err := q.Where("author_id IS NOT NULL").Distinct().Pluck("author_id", &authorIDs).Error; err != nil {
		return nil, err
	}
	if len(authorIDs) == 0 {
		return nil, nil
	}
	db, err := sites.DB(ctx)
	if err != nil {
		return nil, err
	}
	var users []models.User
	err = db.Where("user_id IN ? AND status = ?", authorIDs, models.StatusActive).Order("username ASC").Find(&users).Error
	return users, err
}
