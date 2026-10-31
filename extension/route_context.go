package extension

import "strings"

// PathSuffix returns the path remainder after a route pattern ending in /*.
func PathSuffix(routePath, requestPath string) string {
	return MountPathSuffix(routePath, requestPath)
}

// MountPathSuffix returns the subpath under a mount route pattern.
// Supports explicit wildcards (/jobs/*) and mount base paths (/jobs).
func MountPathSuffix(routePath, requestPath string) string {
	routePath = strings.TrimSpace(routePath)
	requestPath = strings.TrimSpace(requestPath)
	if routePath == "" || requestPath == "" {
		return ""
	}
	if strings.HasSuffix(routePath, "/*") {
		prefix := strings.TrimSuffix(routePath, "/*")
		if requestPath == prefix {
			return ""
		}
		if strings.HasPrefix(requestPath, prefix+"/") {
			return strings.TrimPrefix(requestPath, prefix+"/")
		}
		return ""
	}
	base := strings.TrimRight(routePath, "/")
	if requestPath == base {
		return ""
	}
	if strings.HasPrefix(requestPath, base+"/") {
		return strings.TrimPrefix(requestPath, base+"/")
	}
	return ""
}

// PathSegments splits the request URL path into non-empty segments.
func PathSegments(requestURL string) []string {
	path := requestURL
	if i := strings.Index(path, "://"); i >= 0 {
		if rest := strings.Index(path[i+3:], "/"); rest >= 0 {
			path = path[i+3+rest:]
		} else {
			return nil
		}
	}
	if q := strings.Index(path, "?"); q >= 0 {
		path = path[:q]
	}
	path = strings.Trim(path, "/")
	if path == "" {
		return nil
	}
	return strings.Split(path, "/")
}

// WirePathSuffix returns path_suffix from a wire request (requires RoutePath).
func WirePathSuffix(req WireRequest) string {
	if req.PathSuffix != "" {
		return req.PathSuffix
	}
	if req.RoutePath != "" && req.URL != "" {
		u := req.URL
		if i := strings.Index(u, "://"); i >= 0 {
			if rest := strings.Index(u[i+3:], "/"); rest >= 0 {
				u = u[i+3+rest:]
			}
		}
		if q := strings.Index(u, "?"); q >= 0 {
			u = u[:q]
		}
		return MountPathSuffix(req.RoutePath, u)
	}
	return ""
}
