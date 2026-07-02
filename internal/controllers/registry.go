package controllers

import (
	"sort"
	"sync"
)

// ActionDefinition describes one handler on a controller.
type ActionDefinition struct {
	ID           string
	Title        string
	Methods      []string
	DefaultPath  string
	RequireAuth  bool
	RequireGuest bool
	AllowUnverified bool // skip validated check (verify/reset flows)
}

// Definition is listed in admin when creating controller routes.
type Definition struct {
	ID          string
	Title       string
	Description string
	Actions     []ActionDefinition
}

// Controller handles a registered action.
type Controller interface {
	Handle(ctx *Context, actionID string) Result
}

var (
	mu    sync.RWMutex
	defs  = map[string]Definition{}
	impls = map[string]Controller{}
)

// Register adds a built-in frontend controller.
func Register(def Definition, c Controller) {
	mu.Lock()
	defer mu.Unlock()
	defs[def.ID] = def
	impls[def.ID] = c
}

// Definitions returns registered controllers sorted by title.
func Definitions() []Definition {
	mu.RLock()
	defer mu.RUnlock()
	out := make([]Definition, 0, len(defs))
	for _, def := range defs {
		out = append(out, def)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Title == out[j].Title {
			return out[i].ID < out[j].ID
		}
		return out[i].Title < out[j].Title
	})
	return out
}

// Lookup returns a controller definition and implementation.
func Lookup(id string) (Definition, Controller, bool) {
	mu.RLock()
	defer mu.RUnlock()
	def, ok := defs[id]
	if !ok {
		return Definition{}, nil, false
	}
	c, ok := impls[id]
	return def, c, ok
}

// LookupAction returns an action on a controller.
func LookupAction(controllerID, actionID string) (ActionDefinition, bool) {
	def, _, ok := Lookup(controllerID)
	if !ok {
		return ActionDefinition{}, false
	}
	for _, action := range def.Actions {
		if action.ID == actionID {
			return action, true
		}
	}
	return ActionDefinition{}, false
}

// MethodAllowed reports whether an HTTP method may invoke the action.
func MethodAllowed(action ActionDefinition, method string) bool {
	if len(action.Methods) == 0 {
		return true
	}
	for _, m := range action.Methods {
		if m == method {
			return true
		}
	}
	return false
}
