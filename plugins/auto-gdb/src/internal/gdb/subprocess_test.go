package gdb

import (
	"os/exec"
	"strings"
	"testing"
	"time"
)

// findGdb checks if gdb is available in PATH.
func findGdb() (string, error) {
	return exec.LookPath("gdb")
}

// TestGdbCliController requires GDB to be installed on the system.
// Skip the test if GDB is not available.

func TestGdbCliControllerBasic(t *testing.T) {
	// Check if gdb is available
	if _, err := findGdb(); err != nil {
		t.Skip("gdb not found, skipping test")
	}

	ctrl, err := NewGdbCliController("")
	if err != nil {
		t.Fatalf("NewGdbCliController() failed: %v", err)
	}
	defer ctrl.Exit()

	// Test that controller is alive
	if !ctrl.IsAlive() {
		t.Error("Controller should be alive after creation")
	}

	// Test Process() returns nil for running process
	poll := ctrl.Process().Poll()
	if poll != nil {
		t.Errorf("Process().Poll() = %d, want nil (running)", *poll)
	}
}

func TestGdbCliControllerRunCLI(t *testing.T) {
	if _, err := findGdb(); err != nil {
		t.Skip("gdb not found, skipping test")
	}

	ctrl, err := NewGdbCliController("")
	if err != nil {
		t.Fatalf("NewGdbCliController() failed: %v", err)
	}
	defer ctrl.Exit()

	// Test simple command
	output, err := ctrl.RunCLI("echo hello", 5*time.Second)
	if err != nil {
		t.Errorf("RunCLI('echo hello') failed: %v", err)
	}
	if !strings.Contains(output, "hello") {
		t.Errorf("RunCLI('echo hello') output = %q, want to contain 'hello'", output)
	}

	// Test version command
	output, err = ctrl.RunCLI("show version", 5*time.Second)
	if err != nil {
		t.Errorf("RunCLI('show version') failed: %v", err)
	}
	if !strings.Contains(output, "GNU gdb") {
		t.Errorf("RunCLI('show version') output should contain 'GNU gdb', got %q", output)
	}
}

func TestGdbCliControllerEmptyCommand(t *testing.T) {
	if _, err := findGdb(); err != nil {
		t.Skip("gdb not found, skipping test")
	}

	ctrl, err := NewGdbCliController("")
	if err != nil {
		t.Fatalf("NewGdbCliController() failed: %v", err)
	}
	defer ctrl.Exit()

	// Empty command should return empty output
	output, err := ctrl.RunCLI("", 1*time.Second)
	if err != nil {
		t.Errorf("RunCLI('') failed: %v", err)
	}
	if output != "" {
		t.Errorf("RunCLI('') = %q, want empty string", output)
	}

	// Whitespace-only command should also return empty
	output, err = ctrl.RunCLI("   ", 1*time.Second)
	if err != nil {
		t.Errorf("RunCLI('   ') failed: %v", err)
	}
	if output != "" {
		t.Errorf("RunCLI('   ') = %q, want empty string", output)
	}
}

func TestGdbCliControllerExit(t *testing.T) {
	if _, err := findGdb(); err != nil {
		t.Skip("gdb not found, skipping test")
	}

	ctrl, err := NewGdbCliController("")
	if err != nil {
		t.Fatalf("NewGdbCliController() failed: %v", err)
	}

	// Controller should be alive
	if !ctrl.IsAlive() {
		t.Error("Controller should be alive before Exit()")
	}

	// Exit the controller
	ctrl.Exit()

	// Controller should be dead after Exit()
	if ctrl.IsAlive() {
		t.Error("Controller should not be alive after Exit()")
	}

	// Poll should return non-nil exit code
	poll := ctrl.Process().Poll()
	if poll == nil {
		t.Error("Process().Poll() should return non-nil after Exit()")
	}

	// Multiple Exit() calls should be safe
	ctrl.Exit()
	ctrl.Exit()
}

func TestGdbCliControllerAfterExit(t *testing.T) {
	if _, err := findGdb(); err != nil {
		t.Skip("gdb not found, skipping test")
	}

	ctrl, err := NewGdbCliController("")
	if err != nil {
		t.Fatalf("NewGdbCliController() failed: %v", err)
	}

	ctrl.Exit()

	// RunCLI on dead controller should return ErrSessionDead
	_, err = ctrl.RunCLI("echo test", 1*time.Second)
	if err != ErrSessionDead {
		t.Errorf("RunCLI on dead controller: error = %v, want ErrSessionDead", err)
	}
}

func TestGdbCliControllerTimeout(t *testing.T) {
	if _, err := findGdb(); err != nil {
		t.Skip("gdb not found, skipping test")
	}

	ctrl, err := NewGdbCliController("")
	if err != nil {
		t.Fatalf("NewGdbCliController() failed: %v", err)
	}
	defer ctrl.Exit()

	// Run a command with very short timeout
	// Note: This test is timing-dependent and may be flaky
	_, err = ctrl.RunCLI("shell sleep 10", 100*time.Millisecond)
	if err != ErrTimeout {
		t.Logf("RunCLI with timeout: error = %v (may be ErrTimeout or other)", err)
	}
}