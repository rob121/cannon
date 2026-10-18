package sites

import (
	"context"
	"strings"

	"github.com/rob121/cannon/internal/models"
	"github.com/rob121/cannon/internal/routemeta"
)

const fallbackDefaultRoutePath = "/"

// DefaultRoutePath returns the path of the active default site route, or "/" when unset.
// Wildcard controller routes (ending in /*) are resolved to a concrete path using route
// metadata slugs when present, so Home links never point at a literal "/*" pattern.
func DefaultRoutePath(ctx context.Context) string {
	db, err := DB(ctx)
	if err != nil {
		return fallbackDefaultRoutePath
	}
	var route models.Route
	err = db.Where("is_default = ? AND status = ?", true, models.StatusActive).First(&route).Error
	if err != nil || strings.TrimSpace(route.Path) == "" {
		return fallbackDefaultRoutePath
	}
	return ConcreteRoutePath(route)
}

// ConcreteRoutePath turns a stored route path into a browsable URL path.
func ConcreteRoutePath(route models.Route) string {
	path := strings.TrimSpace(route.Path)
	if path == "" {
		return fallbackDefaultRoutePath
	}
	if !strings.HasSuffix(path, "/*") {
		return path
	}
	prefix := strings.TrimSuffix(path, "/*")
	slugKey := ""
	switch strings.TrimSpace(route.ControllerAction) {
	case "category":
		slugKey = "category_slug"
	case "item", "edit":
		slugKey = "item_slug"
	case "tag":
		slugKey = "tag_slug"
	case "author":
		slugKey = "author_key"
	case "feed":
		return concreteFeedPath(prefix, route.Metadata)
	}
	if slugKey != "" {
		if slug := strings.Trim(routemeta.MetadataString(route.Metadata, slugKey), "/"); slug != "" {
			if prefix == "" {
				return "/" + slug
			}
			return prefix + "/" + slug
		}
	}
	if prefix == "" {
		return fallbackDefaultRoutePath
	}
	return prefix
}

func concreteFeedPath(prefix, metadata string) string {
	kind := strings.TrimSpace(routemeta.MetadataString(metadata, "feed_kind"))
	if kind == "" {
		kind = "global"
	}
	base := prefix
	if base == "" {
		base = "/content/feed"
	}
	switch kind {
	case "category":
		if slug := strings.Trim(routemeta.MetadataString(metadata, "category_slug"), "/"); slug != "" {
			return base + "/category/" + slug
		}
	case "tag":
		if slug := strings.Trim(routemeta.MetadataString(metadata, "tag_slug"), "/"); slug != "" {
			return base + "/tag/" + slug
		}
	case "author":
		if key := strings.Trim(routemeta.MetadataString(metadata, "author_key"), "/"); key != "" {
			return base + "/author/" + key
		}
	}
	return base + "/global"
}
