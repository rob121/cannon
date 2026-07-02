package extension

import "strconv"

// PageData returns a value from the route placement metadata on the wire request.
func PageData(req WireRequest, key string) (any, bool) {
	if req.PageData == nil {
		return nil, false
	}
	v, ok := req.PageData[key]
	return v, ok
}

// PageDataString returns a string metadata value.
func PageDataString(req WireRequest, key string) string {
	v, ok := PageData(req, key)
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

// PageDataInt returns an integer metadata value.
func PageDataInt(req WireRequest, key string) (int, bool) {
	v, ok := PageData(req, key)
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
