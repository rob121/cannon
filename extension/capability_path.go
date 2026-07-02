package extension

import "strings"

func normalizeCapabilityPath(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return ""
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	return strings.TrimRight(path, "/")
}

func (s *Server) pageBase() string {
	if p := normalizeCapabilityPath(s.pagePath); p != "" {
		return p
	}
	return defaultPagePath
}

func (s *Server) blockBase() string {
	if p := normalizeCapabilityPath(s.blockPath); p != "" {
		return p
	}
	return defaultBlockPath
}

func (s *Server) endpointBase() string {
	if p := normalizeCapabilityPath(s.endpointPath); p != "" {
		return p
	}
	return defaultEndpointPath
}

func (s *Server) dataBase() string {
	if p := normalizeCapabilityPath(s.dataPath); p != "" {
		return p
	}
	return defaultDataPath
}
