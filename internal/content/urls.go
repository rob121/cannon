package content

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/rob121/cannon/internal/models"
)

// ItemURL returns the frontend path for an item slug in the default locale layout.
func ItemURL(slug string) string {
	return LocalizedPath(context.Background(), "/content/item/"+slug)
}

// ItemURLForContext returns a locale-aware frontend path for an item slug.
func ItemURLForContext(ctx context.Context, slug string) string {
	return LocalizedPath(ctx, "/content/item/"+strings.TrimPrefix(strings.TrimSpace(slug), "/"))
}

// CategoryURL returns the frontend path for a category slug.
func CategoryURL(slug string) string {
	return LocalizedPath(context.Background(), "/content/category/"+slug)
}

// CategoryURLForContext returns a locale-aware frontend path for a category slug.
func CategoryURLForContext(ctx context.Context, slug string) string {
	return LocalizedPath(ctx, "/content/category/"+strings.TrimPrefix(strings.TrimSpace(slug), "/"))
}

// TagURL returns the frontend path for a tag slug.
func TagURL(slug string) string {
	return LocalizedPath(context.Background(), "/content/tag/"+slug)
}

// TagURLForContext returns a locale-aware frontend path for a tag slug.
func TagURLForContext(ctx context.Context, slug string) string {
	return LocalizedPath(ctx, "/content/tag/"+strings.TrimPrefix(strings.TrimSpace(slug), "/"))
}

// AuthorURL returns the frontend path for an author key (username or id).
func AuthorURL(key string) string {
	return LocalizedPath(context.Background(), "/content/author/"+key)
}

// AuthorURLForContext returns a locale-aware frontend path for an author key.
func AuthorURLForContext(ctx context.Context, key string) string {
	return LocalizedPath(ctx, "/content/author/"+strings.TrimPrefix(strings.TrimSpace(key), "/"))
}

// AuthorKeyFromUser returns the preferred author URL segment for a user.
func AuthorKeyFromUser(user *models.User) string {
	if user == nil {
		return ""
	}
	if user.Username != "" {
		return user.Username
	}
	return strconv.FormatUint(uint64(user.UserID), 10)
}

// SearchURL returns the content search path with an optional query.
func SearchURL(query string) string {
	return SearchURLForContext(context.Background(), query)
}

// SearchURLForContext returns a locale-aware search path with an optional query.
func SearchURLForContext(ctx context.Context, query string) string {
	path := LocalizedPath(ctx, "/content/search")
	if query == "" {
		return path
	}
	return fmt.Sprintf("%s?q=%s", path, url.QueryEscape(query))
}

// FeaturedURL returns the frontend path for the featured items listing.
func FeaturedURL() string {
	return LocalizedPath(context.Background(), "/content/featured")
}

// FeaturedURLForContext returns a locale-aware featured listing path.
func FeaturedURLForContext(ctx context.Context) string {
	return LocalizedPath(ctx, "/content/featured")
}
