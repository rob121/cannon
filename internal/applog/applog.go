package applog

import (
	"log"
	"strings"
	"sync/atomic"
)

// Level controls which messages are emitted.
type Level int32

const (
	LevelDebug Level = iota
	LevelInfo
	LevelWarning
	LevelError
	LevelNone
)

var currentLevel atomic.Int32

func init() {
	currentLevel.Store(int32(LevelInfo))
}

// ParseLevel converts a configuration string to a log level.
func ParseLevel(raw string) Level {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "debug":
		return LevelDebug
	case "warning", "warn":
		return LevelWarning
	case "error":
		return LevelError
	case "none", "off":
		return LevelNone
	default:
		return LevelInfo
	}
}

// SetLevel sets the global minimum log level.
func SetLevel(level Level) {
	currentLevel.Store(int32(level))
}

// LevelName returns the configured level name.
func LevelName() string {
	switch Level(currentLevel.Load()) {
	case LevelDebug:
		return "debug"
	case LevelWarning:
		return "warning"
	case LevelError:
		return "error"
	case LevelNone:
		return "none"
	default:
		return "info"
	}
}

func enabled(level Level) bool {
	return level >= Level(currentLevel.Load()) && Level(currentLevel.Load()) != LevelNone
}

// Debugf logs at debug level.
func Debugf(format string, args ...any) {
	if enabled(LevelDebug) {
		log.Printf(format, args...)
	}
}

// Infof logs at info level.
func Infof(format string, args ...any) {
	if enabled(LevelInfo) {
		log.Printf(format, args...)
	}
}

// Warningf logs at warning level.
func Warningf(format string, args ...any) {
	if enabled(LevelWarning) {
		log.Printf(format, args...)
	}
}

// Errorf logs at error level.
func Errorf(format string, args ...any) {
	if enabled(LevelError) {
		log.Printf(format, args...)
	}
}
