package debug

import (
	"fmt"
	"os"
	"runtime/debug"
	"sync"
	"time"
)

const defaultLogPath = "/tmp/.why-debug.log"

var (
	enabled bool
	logFile *os.File
	mu      sync.Mutex
)

// Init checks WHY_DEBUG and opens the log file if enabled.
// "0" or unset = disabled. "1"/"true" = log to /tmp/.why-debug.log.
// Any other value is treated as a file path.
func Init() {
	val := os.Getenv("WHY_DEBUG")
	if val == "" || val == "0" {
		return
	}
	path := defaultLogPath
	if val != "1" && val != "true" {
		path = val
	}
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return
	}
	logFile = f
	enabled = true
}

// Log writes a timestamped line to the debug log. No-op when disabled.
func Log(format string, args ...any) {
	if !enabled {
		return
	}
	mu.Lock()
	defer mu.Unlock()
	ts := time.Now().Format("2006-01-02T15:04:05.000")
	msg := fmt.Sprintf(format, args...)
	fmt.Fprintf(logFile, "%s %s\n", ts, msg)
}

// CaptureStderr redirects os.Stderr so stray output (from CGO, Go runtime,
// cobra, etc.) doesn't reach Claude Code. When debug is enabled, stderr goes
// to the log file; otherwise to /dev/null.
func CaptureStderr() {
	if enabled && logFile != nil {
		os.Stderr = logFile
	} else {
		if devNull, err := os.Open(os.DevNull); err == nil {
			os.Stderr = devNull
		}
	}
}

// Close closes the log file if open.
func Close() {
	if logFile != nil {
		logFile.Close()
	}
}

// Stack returns the current goroutine's stack trace.
func Stack() []byte {
	return debug.Stack()
}
