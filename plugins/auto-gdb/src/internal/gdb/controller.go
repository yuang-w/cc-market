// Package gdb provides GDB process and socket controllers.
package gdb

import (
	"errors"
	"regexp"
	"time"
)

// Default timeout for GDB operations.
const DefaultTimeout = 15 * time.Second

// Errors returned by controllers.
var (
	ErrTimeout     = errors.New("timeout waiting for GDB response")
	ErrNoSession   = errors.New("no GDB session")
	ErrSessionDead = errors.New("GDB session is dead")
)

// BridgeError wraps an error from gdb_bridge with optional output.
type BridgeError struct {
	Message string
	Output  string
}

func (e *BridgeError) Error() string { return e.Message }

// ANSI escape sequence pattern.
var ansiRe = regexp.MustCompile(`\x1b\[[0-9;]*[a-zA-Z]`)

// StripANSI removes ANSI escape sequences from text.
func StripANSI(s string) string {
	return ansiRe.ReplaceAllString(s, "")
}

// Controller defines the interface for GDB backends.
type Controller interface {
	// RunCLI executes one GDB CLI command and returns the output.
	RunCLI(command string, timeout time.Duration) (string, error)
	// Exit shuts down the GDB session.
	Exit()
	// IsAlive returns true if the GDB session is still running.
	IsAlive() bool
	// Process returns a process-like object for session status checks.
	Process() ProcessStatus
}

// ProcessStatus mimics subprocess poll behavior.
type ProcessStatus interface {
	Poll() *int // nil if running, exit code if dead
}