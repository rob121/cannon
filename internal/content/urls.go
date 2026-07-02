package content

import (
	"fmt"
	"net/url"
	"strconv"

	"github.com/rob121/cannon/internal/models"
)

// ItemURL returns the frontend path for an item slug.
func ItemURL(slug string) string {
	return "/content/item/" + slug
}

// CategoryURL returns the frontend path for a category slug.
func CategoryURL(slug string) string {
	return "/content/category/" + slug
}

// TagURL returns the frontend path for a tag slug.
func TagURL(slug string) string {
	return "/content/tag/" + slug
}

// AuthorURL returns the frontend path for an author key (username or id).
func AuthorURL(key string) string {
	return "/content/author/" + key
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
	if query == "" {
		return "/content/search"
	}
	return fmt.Sprintf("/content/search?q=%s", url.QueryEscape(query))
}

// FeaturedURL returns the frontend path for the featured items listing.
func FeaturedURL() string {
	return "/content/featured"
}
