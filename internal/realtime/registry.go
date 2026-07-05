package realtime

import "sync"

var (
	hubMu sync.RWMutex
	hub   *Hub
)

// SetHub registers the active realtime hub for the process.
func SetHub(h *Hub) {
	hubMu.Lock()
	defer hubMu.Unlock()
	hub = h
}

// ActiveHub returns the active realtime hub, if any.
func ActiveHub() *Hub {
	hubMu.RLock()
	defer hubMu.RUnlock()
	return hub
}
