package gdb

import (
	"testing"
	"time"
)

func TestStripANSI(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "no ANSI codes",
			input:    "Hello, World!",
			expected: "Hello, World!",
		},
		{
			name:     "color codes",
			input:    "\x1b[31mRed text\x1b[0m",
			expected: "Red text",
		},
		{
			name:     "multiple color codes",
			input:    "\x1b[1;32mBold Green\x1b[0m \x1b[33mYellow\x1b[0m",
			expected: "Bold Green Yellow",
		},
		{
			name:     "cursor movement",
			input:    "\x1b[2J\x1b[HClear screen",
			expected: "Clear screen",
		},
		{
			name:     "mixed content",
			input:    "Prefix \x1b[31mcolored\x1b[0m suffix",
			expected: "Prefix colored suffix",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := StripANSI(tt.input)
			if result != tt.expected {
				t.Errorf("StripANSI(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestBridgeError(t *testing.T) {
	err := &BridgeError{
		Message: "test error",
		Output:  "partial output",
	}

	if err.Error() != "test error" {
		t.Errorf("BridgeError.Error() = %q, want %q", err.Error(), "test error")
	}

	if err.Message != "test error" {
		t.Errorf("BridgeError.Message = %q, want %q", err.Message, "test error")
	}

	if err.Output != "partial output" {
		t.Errorf("BridgeError.Output = %q, want %q", err.Output, "partial output")
	}
}

func TestErrors(t *testing.T) {
	tests := []struct {
		name   string
		err    error
		msg    string
	}{
		{"ErrTimeout", ErrTimeout, "timeout waiting for GDB response"},
		{"ErrNoSession", ErrNoSession, "no GDB session"},
		{"ErrSessionDead", ErrSessionDead, "GDB session is dead"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.err.Error() != tt.msg {
				t.Errorf("%s.Error() = %q, want %q", tt.name, tt.err.Error(), tt.msg)
			}
		})
	}
}

func TestDefaultTimeout(t *testing.T) {
	if DefaultTimeout != 15*time.Second {
		t.Errorf("DefaultTimeout = %v, want %v", DefaultTimeout, 15*time.Second)
	}
}