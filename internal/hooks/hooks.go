package hooks

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"sync"
)

// ErrAborted is returned when a hook listener blocks the operation.
var ErrAborted = errors.New("hook aborted")

// Listener handles an in-process hook event.
type Listener func(ctx context.Context, e *Event) (*Result, error)

// Event carries hook name and mutable arguments (Joomla-style plugin event).
type Event struct {
	Name      string
	Arguments map[string]any
	Request   *http.Request
}

// Result is returned by a listener to modify or stop propagation.
type Result struct {
	Arguments map[string]any
	Stop      bool
}

type registry struct {
	mu        sync.RWMutex
	listeners map[string][]Listener
}

var defaultRegistry = &registry{listeners: map[string][]Listener{}}

// Register adds an in-process listener for a hook event.
func Register(event string, fn Listener) {
	defaultRegistry.mu.Lock()
	defer defaultRegistry.mu.Unlock()
	defaultRegistry.listeners[event] = append(defaultRegistry.listeners[event], fn)
}

// Clear removes all in-process listeners (for tests).
func Clear() {
	defaultRegistry.mu.Lock()
	defer defaultRegistry.mu.Unlock()
	defaultRegistry.listeners = map[string][]Listener{}
}

// FireLocal runs registered in-process listeners only.
func FireLocal(ctx context.Context, r *http.Request, event string, args map[string]any) (map[string]any, bool, error) {
	if args == nil {
		args = map[string]any{}
	}
	defaultRegistry.mu.RLock()
	listeners := append([]Listener(nil), defaultRegistry.listeners[event]...)
	defaultRegistry.mu.RUnlock()

	e := &Event{Name: event, Arguments: args, Request: r}
	for _, fn := range listeners {
		out, err := fn(ctx, e)
		if err != nil {
			return args, false, err
		}
		if out == nil {
			continue
		}
		if len(out.Arguments) > 0 {
			args = mergeArgs(args, out.Arguments)
			e.Arguments = args
		}
		if out.Stop {
			return args, true, nil
		}
		if blocked, msg := loginBlocked(args); blocked {
			if msg != "" {
				return args, false, fmt.Errorf("%w: %s", ErrAborted, msg)
			}
			return args, false, ErrAborted
		}
	}
	return args, false, nil
}

// WrapAbort returns an error for hook abort with an optional message.
func WrapAbort(message string) error {
	if message == "" {
		return ErrAborted
	}
	return fmt.Errorf("%w: %s", ErrAborted, message)
}

func loginBlocked(args map[string]any) (bool, string) {
	v, ok := args["allowed"].(bool)
	if ok && !v {
		msg, _ := args["error"].(string)
		return true, msg
	}
	return false, ""
}

func mergeArgs(base, patch map[string]any) map[string]any {
	out := map[string]any{}
	for k, v := range base {
		out[k] = v
	}
	for k, v := range patch {
		out[k] = v
	}
	return out
}

// StringArg reads a string argument.
func StringArg(args map[string]any, key string) string {
	v, ok := args[key].(string)
	if !ok || v == "" {
		return ""
	}
	return v
}

// SetString sets a string argument in place.
func SetString(args map[string]any, key, value string) {
	if args == nil {
		return
	}
	args[key] = value
}
