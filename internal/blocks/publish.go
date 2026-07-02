package blocks

import "time"

// PublishVisible reports whether a block is within its publish window.
func PublishVisible(meta Metadata, now time.Time) bool {
	if meta.PublishStart != nil && now.Before(*meta.PublishStart) {
		return false
	}
	if meta.PublishEnd != nil && now.After(*meta.PublishEnd) {
		return false
	}
	return true
}
