package log

import (
	"errors"
	"strings"
)

// Level represents the severity of a log entry.
type Level int

const (
	Trace Level = iota
	Debug
	Info
	Warn
	Error
	Fatal
)

var levelNames = [...]string{
	"TRACE",
	"DEBUG",
	"INFO",
	"WARN",
	"ERROR",
	"FATAL",
}

// String returns the string representation of the level.
func (l Level) String() string {
	if l < Trace || l > Fatal {
		return "UNKNOWN"
	}
	return levelNames[l]
}

// ErrInvalidLevel is returned when parsing an unknown level string.
var ErrInvalidLevel = errors.New("invalid log level")

// ParseLevel parses a string into a Level.
func ParseLevel(s string) (Level, error) {
	switch strings.ToUpper(s) {
	case "TRACE":
		return Trace, nil
	case "DEBUG":
		return Debug, nil
	case "INFO":
		return Info, nil
	case "WARN", "WARNING":
		return Warn, nil
	case "ERROR":
		return Error, nil
	case "FATAL":
		return Fatal, nil
	default:
		return Info, ErrInvalidLevel
	}
}

// Enables returns true if this level allows logging at the given level.
// A level enables logging for itself and all higher severity levels.
func (l Level) Enables(target Level) bool {
	return target >= l
}
