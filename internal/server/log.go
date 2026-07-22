package server

import "strings"

const (
	LevelDebug = "debug"
	LevelInfo  = "info"
	LevelWarn  = "warn"
	LevelError = "error"
)

var levelOrder = map[string]int{
	LevelDebug: 0,
	LevelInfo:  1,
	LevelWarn:  2,
	LevelError: 3,
}

func levelEnabled(minLevel, entryLevel string) bool {
	entry, ok := levelOrder[strings.ToLower(entryLevel)]
	if !ok {
		return true
	}
	min, ok := levelOrder[strings.ToLower(minLevel)]
	if !ok {
		return true
	}
	return entry >= min
}
