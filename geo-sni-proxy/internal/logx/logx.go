package logx

import (
	"log"
	"os"
	"strings"
)

// Package logx provides opt-in logging controlled by GEO_SNI_PROXY_LOG.
// When enabled, Printf/Println proxy to the standard log package.
// Accepted truthy values: "1", "true", "yes", "on", "debug" (case-insensitive).
var enabled bool

func init() {
	val := strings.TrimSpace(os.Getenv("GEO_SNI_PROXY_LOG"))
	if val == "" {
		enabled = false
		return
	}
	switch strings.ToLower(val) {
	case "1", "true", "yes", "on", "debug":
		enabled = true
	default:
		enabled = false
	}
}

// Enabled reports whether logging is currently enabled.
func Enabled() bool { return enabled }

// SetEnabled allows toggling logging at runtime, if needed by callers.
func SetEnabled(v bool) { enabled = v }

// Printf writes a formatted log line when logging is enabled.
func Printf(format string, v ...any) {
	if enabled {
		log.Printf(format, v...)
	}
}

// Println writes a log line when logging is enabled.
func Println(v ...any) {
	if enabled {
		log.Println(v...)
	}
}

