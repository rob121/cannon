package extension

import "strconv"

// EndpointData returns a value from the route placement metadata on the wire request.
func EndpointData(req WireRequest, key string) (any, bool) {
	if req.EndpointData == nil {
		return nil, false
	}
	v, ok := req.EndpointData[key]
	return v, ok
}

// EndpointDataString returns a string metadata value.
func EndpointDataString(req WireRequest, key string) string {
	v, ok := EndpointData(req, key)
	if !ok || v == nil {
		return ""
	}
	switch t := v.(type) {
	case string:
		return t
	default:
		return ""
	}
}

// EndpointDataInt returns an integer metadata value.
func EndpointDataInt(req WireRequest, key string) (int, bool) {
	v, ok := EndpointData(req, key)
	if !ok || v == nil {
		return 0, false
	}
	switch t := v.(type) {
	case float64:
		return int(t), true
	case int:
		return t, true
	case int64:
		return int(t), true
	case string:
		n, err := strconv.Atoi(t)
		if err != nil {
			return 0, false
		}
		return n, true
	default:
		return 0, false
	}
}
