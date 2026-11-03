package router

import (
	"net/url"
	"strings"
)

const queryMatchScoreBoost = 1000

// splitRoutePattern separates a stored route path from an optional query match suffix.
// Example: "/event-registration?id=14" -> ("/event-registration", "id=14").
func splitRoutePattern(pattern string) (pathPart, queryPart string) {
	pattern = strings.TrimSpace(pattern)
	if i := strings.Index(pattern, "?"); i >= 0 {
		return pattern[:i], pattern[i+1:]
	}
	return pattern, ""
}

func routePatternPath(pattern string) string {
	path, _ := splitRoutePattern(pattern)
	return path
}

func queriesMatchExact(patternQuery, requestRawQuery string) bool {
	patternQuery = strings.TrimSpace(patternQuery)
	if patternQuery == "" {
		return requestRawQuery == ""
	}
	pv, err := url.ParseQuery(patternQuery)
	if err != nil {
		return false
	}
	rv, err := url.ParseQuery(requestRawQuery)
	if err != nil {
		return false
	}
	return pv.Encode() == rv.Encode()
}
