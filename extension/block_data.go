package extension

import "strconv"

// BlockData returns a value from the block placement metadata on the wire request.
func BlockData(req WireRequest, key string) (any, bool) {
	if req.BlockData == nil {
		return nil, false
	}
	v, ok := req.BlockData[key]
	return v, ok
}

// BlockDataString returns a string metadata value.
func BlockDataString(req WireRequest, key string) string {
	v, ok := BlockData(req, key)
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

// BlockDataInt returns an integer metadata value.
func BlockDataInt(req WireRequest, key string) (int, bool) {
	v, ok := BlockData(req, key)
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
