package templateengine

import (
	"net/http"
	"reflect"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/rob121/cannon/internal/content"
	"github.com/rob121/cannon/internal/httpx"
	"github.com/rob121/cannon/internal/models"
)

var htmlTagPattern = regexp.MustCompile(`(?s)<[^>]*>`)

// UserName returns a display label for a user model or name parts.
// An optional display override is used when non-empty.
func UserName(args ...any) string {
	if len(args) == 0 {
		return ""
	}
	if len(args) >= 2 {
		if override := strings.TrimSpace(stringValue(args[len(args)-1])); override != "" {
			return override
		}
	}
	switch v := args[0].(type) {
	case models.User:
		return userNameParts(v.GivenName, v.FamilyName, v.Username)
	case *models.User:
		if v == nil {
			return ""
		}
		return userNameParts(v.GivenName, v.FamilyName, v.Username)
	default:
		if len(args) >= 3 {
			return userNameParts(stringValue(args[0]), stringValue(args[1]), stringValue(args[2]))
		}
	}
	rv := reflect.ValueOf(args[0])
	if rv.Kind() == reflect.Pointer {
		if rv.IsNil() {
			return ""
		}
		rv = rv.Elem()
	}
	if rv.Kind() == reflect.Struct {
		given := stringValue(rv.FieldByName("GivenName").Interface())
		family := stringValue(rv.FieldByName("FamilyName").Interface())
		username := stringValue(rv.FieldByName("Username").Interface())
		if given != "" || family != "" || username != "" {
			return userNameParts(given, family, username)
		}
	}
	if rv.Kind() == reflect.Map {
		given := stringValue(rv.MapIndex(reflect.ValueOf("GivenName")).Interface())
		family := stringValue(rv.MapIndex(reflect.ValueOf("FamilyName")).Interface())
		username := stringValue(rv.MapIndex(reflect.ValueOf("Username")).Interface())
		if username == "" {
			username = stringValue(rv.MapIndex(reflect.ValueOf("username")).Interface())
		}
		if given != "" || family != "" || username != "" {
			return userNameParts(given, family, username)
		}
	}
	return strings.TrimSpace(stringValue(args[0]))
}

func userNameParts(given, family, username string) string {
	given = strings.TrimSpace(given)
	family = strings.TrimSpace(family)
	if given != "" || family != "" {
		return strings.TrimSpace(given + " " + family)
	}
	return strings.TrimSpace(username)
}

func stringValue(v any) string {
	if v == nil {
		return ""
	}
	switch s := v.(type) {
	case string:
		return s
	default:
		return strings.TrimSpace(reflect.ValueOf(v).String())
	}
}

// FormatDate formats a timestamp with a Go layout string.
func FormatDate(t *time.Time, layout string) string {
	if t == nil || t.IsZero() || strings.TrimSpace(layout) == "" {
		return ""
	}
	return t.Format(layout)
}

// ItemPublishedAt returns the publish or created timestamp for an item.
func ItemPublishedAt(item models.Item) *time.Time {
	if item.PublishStart != nil && !item.PublishStart.IsZero() {
		return item.PublishStart
	}
	if item.CreatedAt.IsZero() {
		return nil
	}
	return &item.CreatedAt
}

// ItemDate formats an item publish/created date.
func ItemDate(item models.Item, layout string) string {
	return FormatDate(ItemPublishedAt(item), layout)
}

// ItemDateISO formats an item publish/created date for a datetime attribute.
func ItemDateISO(item models.Item) string {
	return ItemDate(item, "2006-01-02")
}

// ItemHasDate reports whether an item has a displayable publish/created date.
func ItemHasDate(item models.Item) bool {
	return ItemPublishedAt(item) != nil
}

// Excerpt renders rich text as a plain-text teaser truncated to maxLen runes.
func Excerpt(src string, maxLen int) string {
	html, err := content.RichTextToHTML(src)
	if err != nil {
		html = src
	}
	plain := strings.TrimSpace(htmlTagPattern.ReplaceAllString(html, " "))
	plain = strings.Join(strings.Fields(plain), " ")
	if plain == "" {
		return ""
	}
	if maxLen <= 0 {
		maxLen = 160
	}
	if utf8.RuneCountInString(plain) <= maxLen {
		return plain
	}
	runes := []rune(plain)
	return string(runes[:maxLen]) + "…"
}

// ThemeAssetURL returns the public URL for a theme asset path.
func ThemeAssetURL(name string) string {
	name = strings.TrimPrefix(strings.TrimSpace(name), "/")
	if name == "" {
		return "/theme/"
	}
	return "/theme/" + name
}

// AbsoluteURL builds an absolute URL for the current request host.
func AbsoluteURL(r *http.Request, path string) string {
	return httpx.AbsoluteURL(r, path)
}

// IsCurrentPath reports whether the request path matches href.
func IsCurrentPath(r *http.Request, href string) bool {
	if r == nil {
		return false
	}
	current := normalizeURLPath(r.URL.Path)
	target := normalizeURLPath(href)
	if target == "" || target == "#" {
		return false
	}
	if current == target {
		return true
	}
	if target != "/" && strings.HasPrefix(current, target+"/") {
		return true
	}
	return false
}

func normalizeURLPath(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return "/"
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	path = strings.TrimSuffix(path, "/")
	if path == "" {
		return "/"
	}
	return path
}
