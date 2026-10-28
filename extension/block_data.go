package extension

import (
	"strconv"
	"strings"
)

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

// BlockDataIntSlice returns integer metadata stored as a JSON array or a single number.
func BlockDataIntSlice(req WireRequest, key string) []int {
	v, ok := BlockData(req, key)
	if !ok || v == nil {
		return nil
	}
	switch typed := v.(type) {
	case []int:
		return append([]int(nil), typed...)
	case []any:
		out := make([]int, 0, len(typed))
		for _, item := range typed {
			if n, ok := coerceInt(item); ok {
				out = append(out, n)
			}
		}
		return out
	default:
		if n, ok := coerceInt(typed); ok {
			return []int{n}
		}
	}
	return nil
}

func coerceInt(v any) (int, bool) {
	switch t := v.(type) {
	case float64:
		return int(t), true
	case int:
		return t, true
	case int64:
		return int(t), true
	case string:
		n, err := strconv.Atoi(strings.TrimSpace(t))
		if err != nil {
			return 0, false
		}
		return n, true
	default:
		return 0, false
	}
}

// BlockDataBool returns a boolean metadata value stored as bool, 0/1, or common strings.
func BlockDataBool(req WireRequest, key string) bool {
	v, ok := BlockData(req, key)
	if !ok || v == nil {
		return false
	}
	switch t := v.(type) {
	case bool:
		return t
	case string:
		switch strings.ToLower(strings.TrimSpace(t)) {
		case "1", "true", "yes", "on":
			return true
		default:
			return false
		}
	default:
		if n, ok := coerceInt(v); ok {
			return n != 0
		}
	}
	return false
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
